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

package analyze_test

import (
	"go/ast"
	"slices"
	"testing"

	"golang.org/x/tools/go/analysis"

	. "fillmore-labs.com/scopeguard/internal/analyze"
)

func TestTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		src    string
		status MoveStatus
		unused int
	}{
		{
			name: "basic_move",
			src: `
				x := 1
				if true {
					_ = x
				}
			`,
			status: MoveAllowed,
			unused: 0,
		},
		{
			name: "shadowed",
			src: `
				y := 1
				x := y
				if true {
					y := "2"
					_ = y
					if true {
						_, _ = x, y
					}
				}
			`,
			status: MoveBlockedShadowed,
			unused: 0,
		},
		{
			name: "typeChange",
			src: `
				var x any
				{
					x = "string"
				}
				x, y := 1, 2
				x = "string"
				_, _ = x, y
			`,
			status: MoveBlockedTypeIncompatible,
			unused: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			fset, f := parseSource(t, tt.src)
			body, fun := firstBody(t, f)

			pkg, info := checkSource(t, fset, []*ast.File{f})

			p := Pass{Pass: &analysis.Pass{
				Fset:      fset,
				Files:     []*ast.File{f},
				TypesInfo: info,
				Pkg:       pkg,
			}}

			scopes := NewScopeIndex(info.Scopes)

			currentFile := NewCurrentFile(fset, f)

			ua := UsageAnalyzer{
				Pass:       p,
				ScopeIndex: scopes,
			}
			ua.Analyzers.Enable(ScopeAnalyzer)

			ta := TargetAnalyzer{
				Pass:          p,
				TargetScope:   NewTargetScope(scopes),
				SafetyChecker: NewSafetyChecker(p.TypesInfo),
				MaxLines:      -1,
				Conservative:  false,
			}

			usage, _ := ua.Analyze(t.Context(), body, fun)
			cm := ta.CollectMoveCandidates(body, currentFile, usage.ScopeRanges)

			// when
			unused := cm.BlockMovesLosingTypeInfo(usage.Usages)

			// then
			mt := cm.SortedMoveTargets(unused, nil)

			// For this test setup, we expect at most one move target relevant to the test case
			// Check if we found *any* target matching our expectation
			expectedStatus := func(m MoveTarget) bool { return m.Status == tt.status }

			idx := slices.IndexFunc(mt, expectedStatus)
			if idx < 0 {
				if len(mt) > 0 {
					t.Errorf("Expected status %q, got %q", tt.status, mt[0].Status)
				} else {
					t.Errorf("Expected status %q, got none", tt.status)
				}

				return
			}

			if got, want := len(mt[idx].Unused), tt.unused; got != want {
				t.Errorf("Expected %d unused variables, got %d", want, got)
			}
		})
	}
}
