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
	"go/token"
	"iter"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"

	"fillmore-labs.com/scopeguard/internal/run"
	"fillmore-labs.com/scopeguard/internal/set"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/diff"
)

// AnalyzerDiagnostic holds the common data extracted for each diagnostic in the graph.
// It transforms [token.Pos] into byte offsets of the concrete file.
type AnalyzerDiagnostic struct {
	Message string
	Edit    string
	Edits   []diff.Edit
	Info    *Info
	Position
	FileSize int
	ID       EditID
	Apply    bool
}

// AllDiagnostics yields one [AnalyzerDiagnostic] per diagnostic across all roots in the graph,
// skipping duplicates.
func AllDiagnostics(graph *checker.Graph) iter.Seq[AnalyzerDiagnostic] {
	return func(yield func(AnalyzerDiagnostic) bool) {
		// Filter duplicate diagnostics in pkg and pkg.test
		seen := set.New[editKey]()

		for _, act := range graph.Roots {
			pkg := act.Package
			fset, pkgPath := pkg.Fset, pkg.PkgPath
			res := act.Result.(run.Result)

			for _, diag := range act.Diagnostics {
				file := fset.File(diag.Pos) // [token.FileSet.File] caches the last file
				if file == nil {
					continue
				}

				pos := file.PositionFor(diag.Pos, false)

				k := editKey{
					Filename: pos.Filename,
					Offset:   pos.Offset,
					Message:  diag.Message,
				}
				if seen.Contains(k) {
					continue // duplicate
				}

				seen.Add(k)

				var (
					id    EditID
					edit  string
					edits []diff.Edit
				)
				if len(diag.SuggestedFixes) > 0 {
					id = k.editID()
					suggestedFix := diag.SuggestedFixes[0]
					edit = suggestedFix.Message
					edits = toDiffEdits(file, suggestedFix.TextEdits)
				}

				if !yield(AnalyzerDiagnostic{
					ID:      id,
					Message: diag.Message,
					Info:    InfoOf(diag.Category),
					Position: Position{
						Package:  pkgPath,
						File:     pos.Filename,
						Function: functionName(res.Processed, diag.Pos),
						Line:     pos.Line,
					},
					FileSize: file.Size(),
					Edit:     edit,
					Edits:    edits,
				}) {
					return
				}
			}
		}
	}
}

const functionNotFound = "Function not found"

// functionName looks up the function name that contains the given position.
func functionName(processed []run.Function, pos token.Pos) string {
	if i, ok := slices.BinarySearchFunc(processed, pos, posInFunc); ok {
		return processed[i].Name.String()
	}

	return functionNotFound
}

func posInFunc(f run.Function, p token.Pos) int {
	switch {
	case p < f.Pos:
		return 1

	case f.End <= p:
		return -1

	default:
		return 0
	}
}

// toDiffEdits converts a slice of [analysis.TextEdit] to [diff.Edit] values expressed as
// byte offsets into the file and sorts them in offset order.
func toDiffEdits(file *token.File, textEdits []analysis.TextEdit) []diff.Edit {
	edits := make([]diff.Edit, 0, len(textEdits))

	for _, edit := range textEdits {
		start := file.Offset(edit.Pos)

		end := start
		if edit.End != token.NoPos {
			end = file.Offset(edit.End)
		}

		edits = append(edits, diff.Edit{Start: start, End: end, New: string(edit.NewText)})
	}

	diff.SortEdits(edits)

	return edits
}
