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

package analyze

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/ast/edge"
	"golang.org/x/tools/go/ast/inspector"
)

// identShadowed checks whether an identifier used in the declaration would be shadowed
// by a later declaration that would make a move unsafe.
//
// Parameters:
//   - c: Cursor pointing to the declaration being considered for moving
//   - declScope: The scope where the declaration currently is
//   - safeScope: The target scope where we want to move the declaration
//
// Returns true if the move would cause a shadowing conflict.
func (p pass) identShadowed(c inspector.Cursor, declScope, safeScope *types.Scope) bool {
	start, end := c.Node().Pos(), c.Node().End()

	for e := range c.Preorder((*ast.Ident)(nil)) {
		// Filter out definitions and field selectors - we only care about identifier uses
		switch kind, _ := e.ParentEdge(); kind {
		case edge.AssignStmt_Lhs,
			edge.Field_Names,
			edge.SelectorExpr_Sel,
			edge.ValueSpec_Names:
			continue
		}

		id, ok := e.Node().(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}

		usedObj, ok := p.pass.TypesInfo.Uses[id]
		if !ok {
			continue
		}

		if usedObj.Pos() > start {
			// Identifier is declared within the moved statement itself, not a use from outside
			continue
		}

		// Walk up the scope chain from safeScope to declScope, looking for shadowing declarations.
		for scope := safeScope; scope != declScope; scope = scope.Parent() {
			if shadowDecl := scope.Lookup(id.Name); shadowDecl != nil && shadowDecl.Pos() < safeScope.Pos() {
				// Found a declaration in an intermediate scope that was defined before
				// the target position, which would shadow the identifier we're using
				return true
			}
		}

		// Would the identifier be shadowed by a later declaration in the same scope?
		// This handles cases like: y := x + 1; x := 2 (can't move y past the redeclaration of x)
		if shadowDecl := declScope.Lookup(id.Name); shadowDecl != nil && shadowDecl != usedObj &&
			// Check whether the redeclaration is after our current statement (x := x is movable)
			// and before our target position
			end < shadowDecl.Pos() && shadowDecl.Pos() < safeScope.Pos() {
			// Found a later redeclaration that would shadow the identifier
			return true
		}
	}

	return false
}
