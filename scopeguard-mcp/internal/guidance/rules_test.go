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

package guidance

import (
	"regexp"
	"testing"

	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
)

func TestDoneStep(t *testing.T) {
	t.Parallel()

	got := DoneStep("test message %d", 123)
	if got.NextTool != DoneToolName {
		t.Errorf("DoneStep() NextTool = %v, want %v", got.NextTool, DoneToolName)
	}

	wantRec := `test message 123`
	if matched, _ := regexp.MatchString(wantRec, got.Recommendation); !matched {
		t.Errorf("DoneStep() Recommendation = %v, want matching %v", got.Recommendation, wantRec)
	}
}

func TestRulesDispatch(t *testing.T) {
	t.Parallel()

	rules := Rules[int]{
		func(_ engine.Facts, c int) (NextStep, bool) {
			if c == 1 {
				return NextStep{NextTool: "tool1"}, true
			}

			return NextStep{}, false
		},
		func(_ engine.Facts, c int) (NextStep, bool) {
			if c == 2 {
				return NextStep{NextTool: "tool2"}, true
			}

			return NextStep{}, false
		},
	}

	tests := []struct {
		name     string
		c        int
		wantTool string
	}{
		{"match first", 1, "tool1"},
		{"match second", 2, "tool2"},
		{"no match", 3, DoneToolName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := rules.Dispatch(engine.Facts{}, tt.c); got.NextTool != tt.wantTool {
				t.Errorf("Rules.Dispatch() NextTool = %v, want %v", got.NextTool, tt.wantTool)
			}
		})
	}
}
