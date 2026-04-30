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

package guidance_test

import (
	"regexp"
	"testing"

	"fillmore-labs.com/scopeguard/internal/config"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
	. "fillmore-labs.com/scopeguard/scopeguard-mcp/internal/guidance"
)

func TestScopeMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		f    engine.Facts
		want string // regex
	}{
		{
			name: "no fixes",
			f:    engine.Facts{Counts: engine.Counts{Total: 0}},
			want: `^No fixes available$`,
		},
		{
			name: "fixes available not applied",
			f: engine.Facts{
				Counts: engine.Counts{Total: 5, Dropped: 0, Applied: 0},
				Filter: config.All,
			},
			want: `^5 fix\(es\) available \(not applied\)$`,
		},
		{
			name: "fixes applied",
			f: engine.Facts{
				Counts: engine.Counts{Total: 5, Dropped: 0, Applied: 5},
				Filter: config.All,
			},
			want: `^5 fix\(es\) applied$`,
		},
		{
			name: "partial applied",
			f: engine.Facts{
				Counts: engine.Counts{Total: 5, Dropped: 0, Applied: 2},
				Filter: config.All,
			},
			want: `^2 fix\(es\) applied; 3 fix\(es\) available \(not applied\)$`,
		},
		{
			name: "with breaking fixes",
			f: engine.Facts{
				Counts: engine.Counts{
					Total:    2,
					Dropped:  0,
					Applied:  0,
					BySafety: map[config.Safety]int{config.Breaking: 1},
				},
				Filter: config.All,
			},
			want: `^2 fix\(es\) available \(not applied\); 1 breaking fix\(es\): treat diffs as scaffold \(see help topic 'breaking'\)$`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ScopeMessage(tt.f, ScopeContext{})
			if matched, _ := regexp.MatchString(tt.want, got); !matched {
				t.Errorf("ScopeMessage() = %v, want matching %v", got, tt.want)
			}
		})
	}
}

func TestScopeRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		f        engine.Facts
		wantTool string
		wantRec  string // regex
	}{
		{
			name:     "no fixes available",
			f:        engine.Facts{Counts: engine.Counts{Total: 0}},
			wantTool: DoneToolName,
			wantRec:  `No scope fixes available`,
		},
		{
			name:     "apply mode all applied",
			f:        engine.Facts{Counts: engine.Counts{Total: 5, Applied: 5}, Mode: engine.ProcessApply},
			wantTool: AnalyzeToolName,
			wantRec:  `^Re-run analyze to confirm fixes were applied correctly\.$`,
		},
		{
			name:     "apply mode with leftovers",
			f:        engine.Facts{Counts: engine.Counts{Total: 5, Applied: 2}, Mode: engine.ProcessApply},
			wantTool: AnalyzeToolName,
			wantRec:  `^Re-run analyze to confirm fixes were applied correctly; 3 fix\(es\) remain\.$`,
		},
		{
			name: "only safe fixes",
			f: engine.Facts{
				Counts: engine.Counts{
					Total:    3,
					Dropped:  0,
					BySafety: map[config.Safety]int{config.Safe: 3},
				},
				Mode: engine.ProcessPreview,
			},
			wantTool: ScopeToolName,
			wantRec:  `Apply 3 safe fix\(es\) with 'apply_safe'`,
		},
		{
			name: "only safe fixes with dropped",
			f: engine.Facts{
				Counts: engine.Counts{
					Total:    5,
					Dropped:  2,
					BySafety: map[config.Safety]int{config.Safe: 5},
				},
				Mode: engine.ProcessPreview,
			},
			wantTool: ScopeToolName,
			wantRec:  `Apply all 5 safe fix\(es\) \(including .*not shown\) with 'apply_safe'`,
		},
		{
			name: "unsafe diffs",
			f: engine.Facts{
				Counts: engine.Counts{
					Total:    2,
					Dropped:  0,
					BySafety: map[config.Safety]int{config.Unsafe: 2},
				},
				Mode: engine.ProcessPreview,
			},
			wantTool: ScopeToolName,
			wantRec:  `Review 2 unsafe diff\(s\) before applying`,
		},
		{
			name: "breaking diffs",
			f: engine.Facts{
				Counts: engine.Counts{
					Total:    1,
					BySafety: map[config.Safety]int{config.Breaking: 1},
				},
				Mode: engine.ProcessPreview,
			},
			wantTool: ScopeToolName,
			wantRec:  `^Review 1 breaking diff\(s\) before applying \(see help topic 'breaking'\)\.$`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ScopeRules.Dispatch(tt.f, ScopeContext{})
			if got.NextTool != tt.wantTool {
				t.Errorf("ScopeRules.Dispatch() NextTool = %v, want %v", got.NextTool, tt.wantTool)
			}

			if matched, _ := regexp.MatchString(tt.wantRec, got.Recommendation); !matched {
				t.Errorf("ScopeRules.Dispatch() Recommendation = %v, want matching %v", got.Recommendation, tt.wantRec)
			}
		})
	}
}
