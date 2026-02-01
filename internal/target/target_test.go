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

package target_test

import (
	"go/ast"
	"slices"
	"testing"

	"golang.org/x/tools/go/analysis"

	"fillmore-labs.com/scopeguard/internal/astutil"
	"fillmore-labs.com/scopeguard/internal/config"
	"fillmore-labs.com/scopeguard/internal/scope"
	. "fillmore-labs.com/scopeguard/internal/target"
	"fillmore-labs.com/scopeguard/internal/target/check"
	"fillmore-labs.com/scopeguard/internal/testsource"
	"fillmore-labs.com/scopeguard/internal/usage"
)

func TestTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		src    string
		status check.MoveStatus
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
			status: check.MoveAllowed,
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
			status: check.MoveBlockedShadowed,
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
			status: check.MoveBlockedTypeIncompatible,
			unused: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			fset, f, fun, body := testsource.Parse(t, tt.src)
			pkg, info := testsource.Check(t, fset, f)

			p := &analysis.Pass{
				Fset:      fset,
				Files:     []*ast.File{f},
				TypesInfo: info,
				Pkg:       pkg,
			}

			scopes := scope.NewIndex(info)

			behavior := config.DefaultBehavior()
			maxlines := -1

			us := usage.New(p, scopes, config.NewBitMask(config.ScopeAnalyzer), behavior)

			ts := New(p, scopes, maxlines, behavior)

			currentFile := astutil.NewCurrentFile(fset, f)

			usageData, _ := us.TrackUsage(t.Context(), body, fun)
			cm := ts.CollectMoveCandidates(body, currentFile, usageData.AllScopeRanges())

			// when
			unused := cm.BlockMovesLosingTypeInfo(usageData.AllDeclarations())

			// then
			mt := cm.SortedMoveTargets(unused, nil)

			// For this test setup, we expect at most one move target relevant to the test case
			// Check if we found *any* target matching our expectation
			expectedStatus := func(m MoveTarget) bool { return m.Status == tt.status }

			idx := slices.IndexFunc(mt, expectedStatus)
			if idx < 0 {
				if len(mt) > 0 {
					t.Errorf("Got status %q, expected %q", mt[0].Status, tt.status)
				} else {
					t.Errorf("Got no status, expected %q", tt.status)
				}

				return
			}

			if got, want := len(mt[idx].Unused), tt.unused; got != want {
				t.Errorf("Got %d unused variables, expected %d", got, want)
			}
		})
	}
}
