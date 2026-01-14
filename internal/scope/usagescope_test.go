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

package scope_test

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/ast/inspector"

	. "fillmore-labs.com/scopeguard/internal/scope"
	"fillmore-labs.com/scopeguard/internal/testsource"
)

func TestShadowing(t *testing.T) {
	t.Parallel()

	tests := [...]struct {
		name     string
		src      string
		varname  string
		boundary ast.Node
	}{
		{
			name:     "simple",
			src:      `x := 1; { x := 2; _ = x }; _ = x`,
			boundary: (*ast.BlockStmt)(nil),
		},
		{
			name:    "no",
			src:     `x := 1; { y := 2; _ = y }; _ = x`,
			varname: "y",
		},
		{
			name: "different_types",
			src:  `x := "string"; { x := 2; _ = x }; _ = x`,
		},
		{
			// Function boundary stops shadowing check in this implementation
			name: "function",
			src:  `x := 1; func() { x := 2; _ = x }(); _ = x`,
		},
		{
			name:     "if",
			src:      `x := 1; if x := 2; x > 0 { _ = x }; _ = x`,
			boundary: (*ast.IfStmt)(nil),
		},
		{
			name:     "if_init",
			src:      `if x := 1; x > 0 { x := 2; _ = x } else { _ = x }`,
			boundary: (*ast.IfStmt)(nil),
		},
		{
			// Type switch has specific handling, usually strict
			name: "type_switch",
			src:  `x := any(1); switch x := x.(type) { case int: _ = x }; _ = x`,
		},
		{
			name:     "case",
			src:      `x := 1; switch { case true: x := 2; _ = x }; _ = x`,
			boundary: (*ast.SwitchStmt)(nil),
		},
		{
			name:     "select",
			src:      `x := 1; c := make(chan int); select { case x := <-c: _ = x }; _ = x`,
			boundary: (*ast.SelectStmt)(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset, f, _, body := testsource.Parse(t, tt.src)
			_, info := testsource.Check(t, fset, f)

			scopes := NewIndex(info.Scopes)
			us := NewUsageScope(scopes)

			selects := NewSelectIndex(body)

			// Let's default to "x".
			varname := "x"
			if tt.varname != "" {
				varname = tt.varname
			}

			// Find the variable that is declared in the innermost scope
			// In our test cases, this is usually near the end or deeply nested.
			v := lastDef(t, info, varname)
			if v == nil {
				t.Fatalf("Could not find target variable %q definition", varname)
			}

			shadowed, gotBoundary := us.Shadowing(selects, v)
			gotShadow, wantShadow := shadowed != nil, tt.boundary != nil

			if gotShadow != wantShadow {
				t.Errorf("Shadowing() = %t, want %t", gotShadow, wantShadow)
			}

			if wantShadow {
				if wantBoundary := findScope(t, tt.boundary, body, v.Pos()).End(); gotBoundary != wantBoundary {
					t.Errorf("Got boundary %d, want %d", gotBoundary, wantBoundary)
				}
			}
		})
	}
}

// lastDef finds the source-order last definition of the variable, which corresponds to the innermost definition in our test cases.
func lastDef(tb testing.TB, info *types.Info, name string) *types.Var {
	tb.Helper()

	var lastVar *types.Var

	for _, obj := range info.Defs {
		v, ok := obj.(*types.Var)
		if !ok {
			continue
		}

		if v.Name() != name {
			continue
		}

		if lastVar != nil && v.Pos() < lastVar.Pos() {
			continue
		}

		lastVar = v
	}

	return lastVar
}

// findScope finds that last node of type typ around pos.
func findScope(tb testing.TB, typ ast.Node, body inspector.Cursor, pos token.Pos) ast.Node {
	tb.Helper()

	if typ == nil {
		return nil
	}

	var endscope ast.Node

	for n := range body.Preorder(typ) {
		if node := n.Node(); node.Pos() <= pos && pos < node.End() {
			endscope = node
		}
	}

	if endscope == nil {
		tb.Fatalf("Could not find node type %T containing variable", typ)
	}

	return endscope
}
