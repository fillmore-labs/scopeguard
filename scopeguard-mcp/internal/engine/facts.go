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

import "fillmore-labs.com/scopeguard/internal/config"

// Counts is the serialized summary of diagnostics returned to API consumers.
type Counts struct {
	ByCategory map[Issue]int         `json:"by_category,omitempty" jsonschema:"Issue counts by category (scope, shadow); only present on multi-category tools (analyze)"`
	BySafety   map[config.Safety]int `json:"by_safety,omitempty"   jsonschema:"Issue counts by safety class (safe, unsafe, breaking)"`
	Total      int                   `json:"total"                 jsonschema:"Total number of matching issues across the analyzed packages, independent of any limit"`
	Dropped    int                   `json:"dropped,omitzero"      jsonschema:"Number of items not returned because the response was truncated to the limit; total = returned + dropped"`
	Applied    int                   `json:"applied"               jsonschema:"Number of edits written to disk. Always 0 for read-only tools (analyze) and for previews"`
}

// Facts holds aggregate counts and routing context for a single tool call.
// Counts is embedded as the serializable part; the remaining fields are
// private routing context used by message and NextStep functions.
type Facts struct {
	Counts

	Mode   ProcessMode
	Filter config.SafetyFilter
}

// AddDiagnostic records one diagnostic into the aggregate counts.
// It tracks totals and per-safety-tier counts. Category tracking (ByCategory)
// is the caller's responsibility for tools that expose it.
func (f *Facts) AddDiagnostic(info *Info) {
	if f.ByCategory != nil {
		f.ByCategory[info.Issue]++
	}

	if f.BySafety == nil {
		f.BySafety = make(map[config.Safety]int)
	}

	f.BySafety[info.Safety]++
}

// Summarize returns the Counts snapshot for wire output, with [Counts.Dropped]
// computed from [Facts.Total] and [Facts.Previewed].
func (f *Facts) Summarize() Counts {
	return f.Counts
}
