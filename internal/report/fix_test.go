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

	tests := [...]struct {
		name        string
		src         string
		needsParens bool
	}{
		{
			name:        "Root",
			src:         `type T struct{}; _ = T{}`,
			needsParens: true,
		},
		{
			name: "CallExpr",
			src:  `type T struct{}; f := func(t T) T { return t }; _ = f(T{})`,
		},
		{
			name:        "Nested CompositeLit",
			src:         `type (U struct{};T struct{F U}); _ = T{F: U{}}`,
			needsParens: true,
		},
		{
			name: "IndexExpr",
			src:  `type T struct{X int}; var a [1]int; _ = a[T{}.X]`,
		},
		{
			name: "SliceExpr",
			src:  `type T struct{X int}; var s []int; _ = s[T{}.X:]`,
		},
		{
			name:        "UnaryExpr",
			src:         `type T struct{}; _ = &T{}`,
			needsParens: true,
		},
		{
			name:        "SelectorExpr",
			src:         `type T struct{F int}; _ = T{}.F`,
			needsParens: true,
		},
		{
			name:        "KeyValueExpr",
			src:         `type (U struct{}; T struct{K U}); _ = T{K: U{}}`,
			needsParens: true,
		},
		{
			name: "Nested CallExpr",
			src:  "type T struct{}; f := func(t T) T { return t }; _ = f(f(T{}))",
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

			if !e.Valid() {
				t.Fatal("Assignment not found")
			}

			if got, want := NeedParen(e), tt.needsParens; got != want {
				t.Errorf("Got NeedParent() = %v, want %v", got, want)
			}
		})
	}
}
