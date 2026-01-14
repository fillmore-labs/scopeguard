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

package report_test

import (
	"go/ast"
	"testing"

	"golang.org/x/tools/go/ast/edge"
	"golang.org/x/tools/go/ast/inspector"

	. "fillmore-labs.com/scopeguard/internal/report"
	"fillmore-labs.com/scopeguard/internal/testsource"
)

func TestNeedParent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		src      string
		expected bool // true = needs parens
	}{
		{
			name:     "Root",
			src:      `type T struct{}; _ = T{}`,
			expected: true,
		},
		{
			name:     "CallExpr",
			src:      `type T struct{}; f := func(t T) T { return t }; _ = f(T{})`,
			expected: false,
		},
		{
			name:     "Nested CompositeLit",
			src:      `type (U struct{};T struct{F U}); _ = T{F: U{}}`,
			expected: true,
		},
		{
			name:     "IndexExpr",
			src:      `type T struct{X int}; var a [1]int; _ = a[T{}.X]`,
			expected: false,
		},
		{
			name:     "SliceExpr",
			src:      `type T struct{X int}; var s []int; _ = s[T{}.X:]`,
			expected: false,
		},
		{
			name:     "UnaryExpr",
			src:      `type T struct{}; _ = &T{}`,
			expected: true,
		},
		{
			name:     "SelectorExpr",
			src:      `type T struct{F int}; _ = T{}.F`,
			expected: true,
		},
		{
			name:     "KeyValueExpr",
			src:      `type (U struct{}; T struct{K U}); _ = T{K: U{}}`,
			expected: true,
		},
		{
			name:     "Nested CallExpr",
			src:      "type T struct{}; f := func(t T) T { return t }; _ = f(f(T{}))",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, _, _, body := testsource.Parse(t, tt.src)

			var e inspector.Cursor

			for a := range body.Preorder((*ast.AssignStmt)(nil)) {
				stmt := a.Node().(*ast.AssignStmt)
				if id, ok := stmt.Lhs[0].(*ast.Ident); ok && id.Name == "_" {
					e = a.ChildAt(edge.AssignStmt_Rhs, 0)
					break
				}
			}

			if e.Inspector() == nil {
				t.Fatal("Assignment not found")
			}

			if got, want := NeedParent(e), tt.expected; got != want {
				t.Errorf("Got NeedParent() = %v, want %v", got, want)
			}
		})
	}
}
