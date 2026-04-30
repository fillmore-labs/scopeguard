// Copyright 2026 Oliver Eikemeier. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"errors"
	"fmt"
	"go/format"
	"io"
	"iter"
	"os"
	"slices"

	"fillmore-labs.com/scopeguard/internal/config"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/diff"
)

// ProcessedEdit is the per-diagnostic record produced by [Process].
// Each tool projects it into its own edit type.
type ProcessedEdit struct {
	Message  string
	Info     *Info
	Edit     string
	Diff     string
	Position Position
	ID       EditID
	Applied  bool
}

// Compare compares two ProcessedEdit instances lexicographically by file path and numerically by line number.
func (a ProcessedEdit) Compare(b ProcessedEdit) int {
	return a.Position.Compare(b.Position)
}

// Process walks all diagnostics at once, groups them by file, applies merged edits and writes each modified file
// back to disk mwhen the mode is not [ProcessPreview]. The returned records are sorted by position.
//
//   - [ProcessPreview]: it never mutates files; all diagnostics are diff-rendered.
//   - [ProcessApply]: applies only edits whose IDs appear in 'apply'; others are diff-rendered.
//   - [ProcessApplySafe]: applies every safe fix; unsafe and breaking fixes are diff-rendered.
func Process(diagnostics iter.Seq[AnalyzerDiagnostic], mode ProcessMode, applyEdits []EditID, summary *Facts) ([]ProcessedEdit, error) {
	batches, err := collectBatches(diagnostics, mode, applyEdits)
	if err != nil {
		return nil, err
	}

	var total int
	for _, b := range batches {
		total += len(b)
	}

	processed := make([]ProcessedEdit, 0, total)

	for name, b := range batches {
		var err error

		processed, err = processFile(processed, name, b)
		if err != nil {
			return nil, err
		}
	}

	summary.Total += len(processed)
	for _, p := range processed {
		info := p.Info
		summary.AddDiagnostic(info)

		if p.Applied {
			summary.Applied++
		}
	}

	slices.SortStableFunc(processed, ProcessedEdit.Compare)

	return processed, nil
}

var errEditIDsRequired = errors.New("mode 'apply' requires explicit IDs")

func collectBatches(diagnostics iter.Seq[AnalyzerDiagnostic], mode ProcessMode, applyEdits []EditID) (map[string][]AnalyzerDiagnostic, error) {
	filter := config.FilterNothing // preview everything outside of applyEdits by default

	var matchedEdits map[EditID]bool

	switch mode {
	case ProcessPreview:

	case ProcessApply:
		if len(applyEdits) == 0 {
			return nil, errEditIDsRequired
		}

		matchedEdits = make(map[EditID]bool, len(applyEdits))
		for _, id := range applyEdits {
			matchedEdits[id] = false
		}

	case ProcessApplySafe:
		filter = config.FilterSafe // do not apply unsafe or breaking edits without explicit request

	default:
		return nil, fmt.Errorf("unknown mode %v", mode)
	}

	batches := make(map[string][]AnalyzerDiagnostic)

	for d := range diagnostics {
		if _, ok := matchedEdits[d.ID]; ok {
			d.Apply = true
			matchedEdits[d.ID] = true
		} else {
			d.Apply = filter.Allowed(d.Info.Safety)
		}

		batches[d.File] = append(batches[d.File], d)
	}

	var unknown []EditID

	for id, matched := range matchedEdits {
		if !matched {
			unknown = append(unknown, id)
		}
	}

	if len(unknown) > 0 {
		slices.Sort(unknown)

		return nil, fmt.Errorf("unknown edit IDs: %v", unknown)
	}

	return batches, nil
}

// processFile opens a file once, computes per-diagnostic unified diffs,
// and/or writes merged edits back, according to mode.
func processFile(processed []ProcessedEdit, name string, batch []AnalyzerDiagnostic) (_ []ProcessedEdit, err error) {
	flag := os.O_RDONLY

	for _, d := range batch {
		if d.Apply {
			flag = os.O_RDWR
			break
		}
	}

	f, err := os.OpenFile(name, flag, 0)
	if err != nil {
		return nil, fmt.Errorf("can't open %q: %w", name, err)
	}

	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("can't close %q: %w", name, cerr)
		}
	}()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("can't read %q: %w", name, err)
	}

	// Reject if the file changed on disk since the analyzer loaded it.
	if size := batch[0].FileSize; len(data) != size {
		return nil, fmt.Errorf("%q changed on disk since load (expected %d bytes, got %d)",
			name, size, len(data))
	}

	src := string(data)

	// Merge the applied edits, render informational diffs for edits that won't be applied.
	var merged []diff.Edit

	processed = slices.Grow(processed, len(batch))

	for _, d := range batch {
		processed = append(processed, ProcessedEdit{
			ID:       d.ID,
			Message:  d.Message,
			Info:     d.Info,
			Edit:     d.Edit,
			Position: d.Position,
		})

		processedEdit := &processed[len(processed)-1]

		edits := d.Edits
		if len(edits) == 0 {
			continue
		}

		// Determine which edits to apply and which to render as informational diffs.
		if d.Apply {
			if m, ok := diff.Merge(merged, edits); ok {
				merged = m
				processedEdit.Applied = true

				continue
			}
		}

		// render diff (preview, not selected, or merge conflict)
		unified, err := renderUnified(src, name, edits)
		if err != nil {
			return nil, err
		}

		processedEdit.Diff = unified
	}

	if len(merged) == 0 {
		return processed, nil
	}

	applied, err := diff.Apply(src, merged)
	if err != nil {
		return nil, fmt.Errorf("internal error: can't apply diff to %q: %w", name, err)
	}

	result := []byte(applied)

	// Prefer 'gofmt' output, but fall back to the raw applied result rather than
	// losing a correct fix if formatting fails.
	if formatted, ferr := format.Source(result); ferr == nil {
		result = formatted
	}

	// An intermediate failure can leave the file truncated; this is
	// acceptable for a developer CLI tool running locally against a
	// version-controlled repository.
	if err := f.Truncate(int64(len(result))); err != nil {
		return nil, fmt.Errorf("can't resize %q: %w", name, err)
	}

	if _, err := f.WriteAt(result, 0); err != nil {
		return nil, fmt.Errorf("can't write %q: %w", name, err)
	}

	return processed, nil
}

var diffContextLines = 5

// renderUnified computes a unified diff for a single diagnostic's edits
// against the given source bytes.
func renderUnified(src, label string, edits []diff.Edit) (string, error) {
	applied, err := diff.Apply(src, edits)
	if err != nil {
		return "", fmt.Errorf("internal error: can't apply diff: %w", err)
	}

	after := applied
	if formatted, err := format.Source([]byte(applied)); err == nil {
		after = string(formatted)
	}

	lines := diff.Lines(src, after)

	unified, err := diff.ToUnified(label, label, src, lines, diffContextLines)
	if err != nil {
		return "", fmt.Errorf("internal error: can't create unified diff: %w", err)
	}

	return unified, nil
}
