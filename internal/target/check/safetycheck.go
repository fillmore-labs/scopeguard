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

package check

import (
	"go/ast"
	"go/types"
	"iter"

	"golang.org/x/tools/go/ast/edge"
	"golang.org/x/tools/go/ast/inspector"
)

// SafetyCheck evaluates a move candidate against safety rules.
func SafetyCheck(info *types.Info, decl inspector.Cursor, declScope, targetScope *types.Scope, identifiers iter.Seq[*ast.Ident]) MoveStatus {
	// Check if identifiers are already declared in the target scope
	if alreadyDeclaredInScope(targetScope, identifiers) {
		return MoveBlockedDeclared
	}

	// Check if moving would cause variables to be shadowed
	if usedIdentifierShadowed(info, decl, declScope, targetScope) {
		return MoveBlockedShadowed
	}

	return MoveAllowed
}

// alreadyDeclaredInScope checks whether any identifier is already declared in the target scope.
func alreadyDeclaredInScope(safeScope *types.Scope, identifiers iter.Seq[*ast.Ident]) bool {
	for id := range identifiers {
		// Check whether the identifier already exists at that level
		if safeScope.Lookup(id.Name) != nil {
			return true
		}
	}

	return false
}

// usedIdentifierShadowed checks whether any identifier used in the declaration would be
// shadowed by a later declaration that would make the move unsafe.
func usedIdentifierShadowed(info *types.Info, decl inspector.Cursor, declScope, safeScope *types.Scope) bool {
	declNode := decl.Node()
	start, end := declNode.Pos(), declNode.End()

	// Track which identifiers we've already checked to avoid redundant work
	checked := make(map[string]struct{})

	// Traverse all identifiers used in the declaration
	for c := range decl.Preorder((*ast.Ident)(nil)) {
		// Filter out definitions and field selectors - we only care about identifier uses
		switch kind, _ := c.ParentEdge(); kind {
		case edge.AssignStmt_Lhs, // Left-hand side of assignment (definition)
			edge.Field_Names,      // Struct field names
			edge.SelectorExpr_Sel, // Right side of dot selector (x.Field)
			edge.ValueSpec_Names:  // Variable declaration names
			continue
		}

		id, ok := c.Node().(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}

		// Skip if we've already checked this identifier
		if _, ok := checked[id.Name]; ok {
			continue
		}

		// Get the object this identifier refers to
		use, ok := info.Uses[id]
		if !ok {
			continue
		}

		// Skip identifiers declared within the statement itself
		// (e.g., in "x, y := f()", x and y are declared here, not uses)
		if use.Pos() > start {
			continue
		}

		// Intermediate scope shadowing
		// Walk up the scope chain from safeScope to declScope, looking for shadowing declarations.
		for scope := safeScope; scope != declScope; scope = scope.Parent() {
			if shadowDecl := scope.Lookup(id.Name); shadowDecl != nil && shadowDecl.Pos() < safeScope.Pos() {
				// Found a declaration in an intermediate scope that was defined before
				// the target position, which would shadow the identifier we're using
				return true
			}
		}

		// Same-scope redeclaration shadowing.
		// This handles cases like: y := x + 1; x := "2" (can't move y past the redeclaration of x)
		if shadowDecl := declScope.Lookup(id.Name); shadowDecl != nil && shadowDecl != use &&
			// Check whether the redeclaration is after our current statement (x := x is movable)
			// and before our target position
			end < shadowDecl.Pos() && shadowDecl.Pos() < safeScope.Pos() {
			// Found a later redeclaration that would shadow the identifier
			return true
		}

		checked[id.Name] = struct{}{}
	}

	return false
}
