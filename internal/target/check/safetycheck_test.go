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

package check_test

import (
	"go/ast"
	"go/types"
	"slices"
	"testing"

	"golang.org/x/tools/go/ast/inspector"

	. "fillmore-labs.com/scopeguard/internal/target/check"
	"fillmore-labs.com/scopeguard/internal/testsource"
)

func TestSafetyCheck(t *testing.T) {
	t.Parallel()

	const targetName = "x"

	tests := [...]struct {
		name string
		src  string
		want MoveStatus
	}{
		{"simple", `var x int; {_ = x}`, MoveAllowed},
		{"declared", `var x int; {x := x; _ = x}`, MoveBlockedDeclared},
		{"declared_2", `x := 0; {y := 0; var x = x; _, _ = x, y}`, MoveBlockedDeclared},
		{"shadowed", `y := 0; x := y; {y := ""; {_, _ = x, y}}`, MoveBlockedShadowed},
		{"not_shadowed", `z := 0; x := z; {y := ""; {_, _, _ = x, y, z}}`, MoveAllowed},
		{"redeclaration", `z := 0; {x := z; z := 1; _ = z; {_ = x}}`, MoveBlockedShadowed},
		{"no_shadow_after", `z := 0; {x := z; {_ = x}; z := 1; _ = z}`, MoveAllowed},
		{"multiple_dependencies", `a, b := 1, 1; x := a + b; {_ = x}`, MoveAllowed},
		{"struct_field", `y := struct{f int}{}; x := y.f; {_ = x}`, MoveAllowed},
		{"shadowed_struct", `y := struct{f int}{}; x := y.f; {y := 0; {_, _ = x, y}}`, MoveBlockedShadowed},
		{"array_index", `y := [1]int{}; x := y[0]; {_ = x}`, MoveAllowed},
		{"shadowed_array", `y := [1]int{}; x := y[0]; {y := 0; {_, _ = x, y}}`, MoveBlockedShadowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset, f, _, body := testsource.Parse(t, tt.src)
			_, info := testsource.Check(t, fset, f)

			decl, declScope, targetScope := prepareScopes(t, info, body, targetName)
			identifiers := slices.Values([]string{targetName})

			if got, want := SafetyCheck(info, decl, declScope, targetScope, identifiers), tt.want; got != want {
				t.Errorf("Expected safety check %q, got %q", want, got)
			}
		})
	}
}

// prepareScopes sets up the scope analysis context for testing FindSafeScope.
//
// It finds the first variable usage.
func prepareScopes(t *testing.T, info *types.Info, body inspector.Cursor, targetName string) (decl inspector.Cursor, declScope, minScope *types.Scope) {
	t.Helper()

	for n := range body.Preorder((*ast.Ident)(nil)) {
		id, ok := n.Node().(*ast.Ident)
		if !ok || id.Name != targetName {
			continue
		}

		if def, ok := info.Defs[id]; ok {
			for d := range n.Enclosing((*ast.AssignStmt)(nil), (*ast.DeclStmt)(nil)) {
				decl = d
				declScope = def.Parent()

				break
			}

			continue
		}

		if _, ok := info.Uses[id]; ok {
			if declScope == nil {
				break
			}

			minScope = declScope.Innermost(id.Pos())

			return decl, declScope, minScope
		}
	}

	t.Fatal("Usage not found")

	return inspector.Cursor{}, nil, nil
}
