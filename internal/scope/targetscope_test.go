// Copyright 2025-2026 Oliver Eikemeier. All Rights Reserved.
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

package scope_test

import (
	"go/ast"
	"go/types"
	"reflect"
	"testing"

	"golang.org/x/tools/go/ast/inspector"

	. "fillmore-labs.com/scopeguard/internal/scope"
	"fillmore-labs.com/scopeguard/internal/testsource"
)

func TestFindSafeScope(t *testing.T) {
	t.Parallel()

	tests := [...]struct {
		name string
		src  string
		want ast.Node
	}{
		{
			name: "simple_block",
			src:  `x := 1; { _ = x }`,
			want: (*ast.BlockStmt)(nil),
		},
		{
			name: "if_block",
			src:  `x := 1; if true { _ = x }`,
			want: (*ast.BlockStmt)(nil),
		},
		{
			name: "for_loop",
			src:  `for x := 0; x < 10; x++ { _ = x }`,
			want: (*ast.ForStmt)(nil),
		},
		{
			name: "for_loop_body",
			src:  `x := 1; for i := 0; i < 10; i++ { _ = x }`,
			want: (*ast.ForStmt)(nil),
		},
		{
			name: "for_loop_init",
			src:  `x := 1; for y := x; y < 10; y++ { }`,
			want: (*ast.ForStmt)(nil),
		},
		{
			name: "for_loop_condition",
			src:  `x := 1; for i := 0; i < x; i++ { }`,
			want: (*ast.ForStmt)(nil),
		},
		{
			name: "range_loop",
			src:  `for x := range 5 { _ = x }`,
			want: (*ast.RangeStmt)(nil),
		},
		{
			name: "range_loop_expr",
			src:  `x := []int{1}; for _, v := range x { _ = v }`,
			want: (*ast.FuncType)(nil),
		},
		{
			name: "nested_blocks",
			src:  `x := 1; { { _ = x } }`,
			want: (*ast.BlockStmt)(nil),
		},
		{
			name: "for_loop_nested_block",
			src:  `x := 1; for i := 0; i < 10; i++ { { _ = x } }`,
			want: (*ast.ForStmt)(nil),
		},
		{
			name: "type_switch",
			src:  `x := any(1); switch x.(type) { case int: }`,
			want: (*ast.TypeSwitchStmt)(nil),
		},
		{
			name: "switch_case_in",
			src:  `x := 1; switch 1 { case x: }`,
			want: (*ast.SwitchStmt)(nil),
		},
		{
			name: "switch_case_out",
			src:  `x := 1; switch { case true: _ = x }`,
			want: (*ast.CaseClause)(nil),
		},
		{
			name: "switch_case_funclit",
			src:  `x := 1; switch 1 { case func() int { return x }(): }`,
			want: (*ast.SwitchStmt)(nil),
		},
		{
			name: "select_case_send",
			src:  `x := 1; ch := make(chan int); select { case ch <- x: }`,
			want: (*ast.FuncType)(nil),
		},
		{
			name: "select_case_recv",
			src:  `x := 1; ch := make(chan int); select { case x = <-ch: _ = x }`,
			want: (*ast.FuncType)(nil),
		},
		{
			name: "select_case_out",
			src:  `x := 1; ch := make(chan int); select { case ch <- 1: _ = x }`,
			want: (*ast.CommClause)(nil),
		},
		{
			name: "select_case_funclit",
			src:  `x := 1; ch := make(chan int); { select { case ch <- func() int { return x }(): } }`,
			want: (*ast.BlockStmt)(nil),
		},
		{
			name: "funclit",
			src:  `x := 1; { _ = func() { _ = x } }`,
			want: (*ast.BlockStmt)(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset, f, _, body := testsource.Parse(t, tt.src)
			_, info := testsource.Check(t, fset, f)

			scopes := NewIndex(info)

			declScope, minScope := prepareScopes(t, info, scopes, body)

			ts := NewTargetScope(scopes)
			safeScope := ts.FindSafeScope(declScope, minScope)
			node := scopes[safeScope]

			if got, want := reflect.TypeOf(node), reflect.TypeOf(tt.want); got != want {
				t.Errorf("Expected %s scope, got %s scope", Name(tt.want), Name(node))
			}
		})
	}
}

// prepareScopes sets up the scope analysis context for testing FindSafeScope.
//
// It finds the first variable usage.
func prepareScopes(t *testing.T, info *types.Info, scopes Index, body inspector.Cursor) (declScope, minScope *types.Scope) {
	t.Helper()

	const targetName = "x"

	for n := range body.Preorder((*ast.Ident)(nil)) {
		id, ok := n.Node().(*ast.Ident)
		if !ok || id.Name != targetName {
			continue
		}

		if use, ok := info.Uses[id]; ok {
			declScope = use.Parent()
			minScope = scopes.Innermost(declScope, id.Pos())

			return declScope, minScope
		}
	}

	t.Fatal("Usage not found")

	return nil, nil
}
