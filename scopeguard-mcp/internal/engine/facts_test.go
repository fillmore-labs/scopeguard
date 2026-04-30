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

package engine_test

import (
	"testing"

	"fillmore-labs.com/scopeguard/internal/config"
	. "fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
)

func TestSummarize(t *testing.T) {
	t.Parallel()

	diags := []AnalyzerDiagnostic{
		{Info: InfoOf("mov"), Position: Position{Package: "example.com/a"}},
		{Info: InfoOf("mov"), Position: Position{Package: "example.com/a"}},
		{Info: InfoOf("mov"), Position: Position{Package: "example.com/b"}},
		{Info: InfoOf("xst"), Position: Position{Package: "example.com/a"}},
		{Info: InfoOf("dec"), Position: Position{Package: "example.com/b"}},
		{Info: InfoOf("uas"), Position: Position{Package: "example.com/a"}},
	}

	tests := []struct {
		wantBySafety   map[config.Safety]int
		wantByCategory map[Issue]int
		name           string
		diags          []AnalyzerDiagnostic
	}{
		{
			name:           "full set",
			diags:          diags,
			wantBySafety:   map[config.Safety]int{config.Safe: 4, config.Unsafe: 1, config.Breaking: 1},
			wantByCategory: map[Issue]int{IssueScope: 5, IssueShadow: 1},
		},
		{
			name:         "empty sequence",
			diags:        nil,
			wantBySafety: map[config.Safety]int{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var s Facts
			s.ByCategory = make(map[Issue]int)

			for _, d := range tc.diags {
				s.AddDiagnostic(d.Info)
			}

			counts := s.Summarize()

			for key, want := range tc.wantBySafety {
				if got := counts.BySafety[key]; got != want {
					t.Errorf("BySafety[%q] = %d, want %d", key, got, want)
				}
			}

			// Ensure empty maps are truly empty (no stray keys).
			if len(tc.wantBySafety) == 0 && len(counts.BySafety) != 0 {
				t.Errorf("BySafety should be empty, got %v", counts.BySafety)
			}

			for key, want := range tc.wantByCategory {
				if got := counts.ByCategory[key]; got != want {
					t.Errorf("ByCategory[%q] = %d, want %d", key, got, want)
				}
			}

			// Ensure empty maps are truly empty (no stray keys).
			if len(tc.wantByCategory) == 0 && len(counts.ByCategory) != 0 {
				t.Errorf("ByCategory should be empty, got %v", counts.ByCategory)
			}
		})
	}
}
