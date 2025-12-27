// Copyright 2025 Oliver Eikemeier. All Rights Reserved.
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
	"go/token"
	"go/types"
	"reflect"
	"testing"

	. "fillmore-labs.com/scopeguard/internal/analyze"
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

			scopes, declScope, minScope := prepareScopes(t, tt.src)

			safeScope := scopes.FindSafeScope(declScope, minScope)
			node := scopes[safeScope]

			if got, want := reflect.TypeOf(node), reflect.TypeOf(tt.want); got != want {
				t.Errorf("Expected %s scope, got %s scope", ScopeName(tt.want), ScopeName(node))
			}
		})
	}
}

// prepareScopes sets up the scope analysis context for testing FindSafeScope.
//
// It parses the source code (wrapped in a function body), builds scope information
// and finds the first variable declaration and its usage.
func prepareScopes(t *testing.T, src string) (scopes ScopeAnalyzer, declScope, minScope *types.Scope) {
	t.Helper()

	fset, f := parseSource(t, src)
	_, info := checkSource(t, fset, []*ast.File{f})

	scopes = NewScopeAnalyzer(info.Scopes)

	fn, decl, use := findUsage(info, f)
	if fn == nil {
		t.Fatal("Usage not found")
	}

	funcScope := info.Scopes[fn]

	// Find the innermost scopes where the variable is declared and used
	declScope = funcScope.Innermost(decl)
	minScope = scopes.Innermost(declScope, use)

	return scopes, declScope, minScope
}

// findUsage locates the function and usage position for testing.
//
// Returns:
//   - fn: The first *ast.FuncType found (the wrapper function created by parseSource)
//   - decl: The position of the declaration of the tracked variable
//   - use: The position of the first usage of the tracked variable
//
// The variable is always declared at function scope.
func findUsage(info *types.Info, root ast.Node) (fn *ast.FuncType, decl, use token.Pos) {
	const targetName = "x"

	for n := range ast.Preorder(root) {
		switch n := n.(type) {
		case *ast.FuncType:
			if fn == nil {
				fn = n
			}

		case *ast.Ident:
			if n.Name != targetName {
				continue
			}

			if _, ok := info.Defs[n]; ok {
				decl = n.Pos()

				continue
			}

			// Find the target variable usage
			if _, ok := info.Uses[n]; !ok {
				continue
			}

			use = n.Pos()

			return fn, decl, use
		}
	}

	return nil, token.NoPos, token.NoPos
}
