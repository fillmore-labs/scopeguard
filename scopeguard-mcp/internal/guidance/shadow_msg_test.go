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

	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
	. "fillmore-labs.com/scopeguard/scopeguard-mcp/internal/guidance"
)

func TestShadowMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		f    engine.Facts
		want string // regex
	}{
		{
			name: "no issues",
			f:    engine.Facts{Counts: engine.Counts{Total: 0}},
			want: `^No issues found$`,
		},
		{
			name: "issues found",
			f:    engine.Facts{Counts: engine.Counts{Total: 3, Dropped: 0}},
			want: `^3 issue\(s\) found$`,
		},
		{
			name: "issues with dropped",
			f:    engine.Facts{Counts: engine.Counts{Total: 5, Dropped: 2}},
			want: `^5 issue\(s\) found; 2 not shown \(see help topic 'limits'\)$`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ShadowMessage(tt.f, ShadowContext{})
			if matched, _ := regexp.MatchString(tt.want, got); !matched {
				t.Errorf("ShadowMessage() = %v, want matching %v", got, tt.want)
			}
		})
	}
}

func TestShadowRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		f        engine.Facts
		wantTool string
		wantRec  string // regex
	}{
		{
			name:     "no results",
			f:        engine.Facts{Counts: engine.Counts{Total: 0}},
			wantTool: DoneToolName,
			wantRec:  `No shadow issues found`,
		},
		{
			name:     "apply mode",
			f:        engine.Facts{Counts: engine.Counts{Total: 1}, Mode: engine.ProcessApply},
			wantTool: AnalyzeToolName,
			wantRec:  `Re-run analyze to confirm renames were applied correctly`,
		},
		{
			name:     "preview mode",
			f:        engine.Facts{Counts: engine.Counts{Total: 3, Dropped: 0}, Mode: engine.ProcessPreview},
			wantTool: ShadowToolName,
			wantRec:  `Build a renames map and call shadow to apply 3 rename\(s\)`,
		},
		{
			name:     "preview mode with dropped",
			f:        engine.Facts{Counts: engine.Counts{Total: 5, Dropped: 2}, Mode: engine.ProcessPreview},
			wantTool: ShadowToolName,
			wantRec:  `Build a renames map and call shadow to apply the 3 shown rename\(s\)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ShadowRules.Dispatch(tt.f, ShadowContext{})
			if got.NextTool != tt.wantTool {
				t.Errorf("ShadowRules.Dispatch() NextTool = %v, want %v", got.NextTool, tt.wantTool)
			}

			if matched, _ := regexp.MatchString(tt.wantRec, got.Recommendation); !matched {
				t.Errorf("ShadowRules.Dispatch() Recommendation = %v, want matching %v", got.Recommendation, tt.wantRec)
			}
		})
	}
}
