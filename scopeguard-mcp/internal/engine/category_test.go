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
	"strings"
	"testing"

	"fillmore-labs.com/scopeguard/internal/category"
	"fillmore-labs.com/scopeguard/internal/config"
	. "fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
)

func TestCategory_Info(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		category   string
		wantReason string
		wantIssue  Issue
		wantSafety config.Safety
	}{
		{
			name:       "MoveBlockedTypeIncompatible",
			category:   "typ",
			wantSafety: config.Breaking,
			wantIssue:  IssueScope,
			wantReason: "Type information lost",
		},
		{
			name:       "CategoryNestedAssignment",
			category:   category.NestedAssignment,
			wantSafety: config.Safe,
			wantIssue:  IssueNestedAssignment,
			wantReason: "Nested reassignment",
		},
		{
			name:       "CategoryShadowed",
			category:   category.Shadowed,
			wantSafety: config.Safe,
			wantIssue:  IssueShadow,
			wantReason: "",
		},
		{
			name:       "Unknown Category",
			category:   "unknown_category",
			wantSafety: config.Unknown,
			wantIssue:  IssueUnknown,
			wantReason: "Internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			info := InfoOf(tt.category)
			if got := info.Safety; got != tt.wantSafety {
				t.Errorf("InfoOf(%q).Safety got %v, want %v", tt.category, got, tt.wantSafety)
			}

			if gotClass := info.Issue; gotClass != tt.wantIssue {
				t.Errorf("InfoOf(%q).Issue got %v, want %v", tt.category, gotClass, tt.wantIssue)
			}

			if gotReason := info.Reason; !strings.HasPrefix(gotReason, tt.wantReason) {
				t.Errorf("InfoOf(%q).Reason got %q, should start with %q", tt.category, gotReason, tt.wantReason)
			}
		})
	}
}
