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

func TestAnalyzeMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		f    engine.Facts
		want string // regex
	}{
		{
			name: "no issues, default tiers",
			f:    engine.Facts{Counts: engine.Counts{Total: 0}, Filter: config.All},
			want: `^No issues found$`,
		},
		{
			name: "no matching issues with safety filter",
			f:    engine.Facts{Counts: engine.Counts{Total: 0}, Filter: config.Safe},
			want: `^No matching issues found \(safety filter active\)$`,
		},
		{
			name: "issues found",
			f: engine.Facts{
				Counts: engine.Counts{
					Total:   5,
					Dropped: 0,
					ByCategory: map[engine.Issue]int{
						engine.IssueScope:  3,
						engine.IssueShadow: 2,
					},
				},
				Filter: config.All,
			},
			want: `^3 scope, 2 shadow issue\(s\) found$`,
		},
		{
			name: "with breaking fixes",
			f: engine.Facts{
				Counts: engine.Counts{
					Total:      1,
					Dropped:    0,
					ByCategory: map[engine.Issue]int{engine.IssueScope: 1},
					BySafety:   map[config.Safety]int{config.Breaking: 1},
				},
				Filter: config.All,
			},
			want: `^1 scope issue\(s\) found; 1 issue\(s\) have compilation-breaking fixes \(see help topic 'breaking'\)$`,
		},
		{
			name: "with dropped",
			f: engine.Facts{
				Counts: engine.Counts{
					Total:      10,
					Dropped:    5,
					ByCategory: map[engine.Issue]int{engine.IssueScope: 10},
				},
				Filter: config.All,
			},
			want: `^10 scope issue\(s\) found; 5 not shown \(see help topic 'limits'\)$`,
		},
		{
			name: "total with empty by-category",
			f: engine.Facts{
				Counts: engine.Counts{Total: 4, Dropped: 0},
				Filter: config.All,
			},
			want: `^4 issue\(s\) found$`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := AnalyzeMessage(tt.f, AnalyzeContext{})
			if matched, _ := regexp.MatchString(tt.want, got); !matched {
				t.Errorf("AnalyzeMessage() = %v, want matching %v", got, tt.want)
			}
		})
	}
}

func TestAnalyzeRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		f        engine.Facts
		wantTool string
		wantRec  string // regex
	}{
		{
			name:     "no issues",
			f:        engine.Facts{Counts: engine.Counts{Total: 0}},
			wantTool: DoneToolName,
			wantRec:  `No issues found`,
		},
		{
			name: "many safe issues",
			f: engine.Facts{
				Counts: engine.Counts{
					Total:    250,
					BySafety: map[config.Safety]int{config.Safe: 250},
				},
			},
			wantTool: ScopeToolName,
			wantRec:  `250 safe issues found: consider a blind safe-apply`,
		},
		{
			name: "many total issues",
			f: engine.Facts{
				Counts: engine.Counts{
					Total: 250,
				},
			},
			wantTool: ScopeToolName,
			wantRec:  `250 issues found: consider function-scoped iteration`,
		},
		{
			name: "scope issues",
			f: engine.Facts{
				Counts: engine.Counts{
					Total:      5,
					ByCategory: map[engine.Issue]int{engine.IssueScope: 5},
				},
			},
			wantTool: ScopeToolName,
			wantRec:  `Preview 5 scope fix\(es\)`,
		},
		{
			name: "shadow issues",
			f: engine.Facts{
				Counts: engine.Counts{
					Total:      2,
					ByCategory: map[engine.Issue]int{engine.IssueShadow: 2},
				},
			},
			wantTool: ShadowToolName,
			wantRec:  `Preview 2 shadow rename\(s\)`,
		},
		{
			name: "manual-review fallthrough",
			f: engine.Facts{
				Counts: engine.Counts{
					Total:      3,
					ByCategory: map[engine.Issue]int{engine.IssueNestedAssignment: 3},
				},
			},
			wantTool: DoneToolName,
			wantRec:  `^3 issue\(s\) require manual review\.$`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := AnalyzeRules.Dispatch(tt.f, AnalyzeContext{})
			if got.NextTool != tt.wantTool {
				t.Errorf("AnalyzeRules.Dispatch() NextTool = %v, want %v", got.NextTool, tt.wantTool)
			}

			if matched, _ := regexp.MatchString(tt.wantRec, got.Recommendation); !matched {
				t.Errorf("AnalyzeRules.Dispatch() Recommendation = %v, want matching %v", got.Recommendation, tt.wantRec)
			}
		})
	}
}
