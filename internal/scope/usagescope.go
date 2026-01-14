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

package scope

import (
	"go/ast"
	"go/token"
	"go/types"
	"maps"
)

// UsageScope determines the usage scope of declared variables.
// It extends ScopeIndex with usage-specific scope analysis.
type UsageScope struct {
	Index
}

// NewUsageScope creates a new [UsageScope] instance.
func NewUsageScope(scopes Index) UsageScope {
	return UsageScope{Index: scopes}
}

// CommonAncestor finds the lowest common ancestor (LCA) of two scopes in the scope tree.
//
//   - declScope: The declaration scope (root of the subtree we're searching)
//   - currentScope: First scope (the current minimum scope)
//   - usageScope: Second scope (scope of the new use we're processing)
func (s UsageScope) CommonAncestor(declScope, currentScope, usageScope *types.Scope) *types.Scope {
	switch usageScope {
	case currentScope, // Same scope as before: no change needed
		declScope: // Tightest possible
		return usageScope
	}

	// Phase 1: Build a path from currentScope to declScope
	// This creates a set of all scopes in the path
	path := maps.Collect(s.ParentScopes(declScope, currentScope))

	// Phase 2: Walk from usageScope to declScope
	// Return the first scope that exists in both paths (the LCA)
	for scope := range s.ParentScopes(declScope, usageScope) {
		if _, ok := path[scope]; ok {
			return scope
		}
	}

	// If we reach here, LCA is the declScope itself
	// This means the two scopes are in completely different branches
	return declScope
}

// Shadowing looks for a shadowed variable in parent scopes.
//
// Parameters:
//   - selects: Index mapping each *[ast.CommClause] to its parent *[ast.SelectStmt]
//   - variable: The variable that may be shadowing another
//
// Returns:
//   - shadowed variable: The outer variable being shadowed (nil if none found)
//   - boundary: The position where the shadowing ends
//
// For nested if statements, the boundary is set to the outermost statement to flag constructs like:
//
//	err := ...()
//	if err != nil {
//	    err := ...() // This shadows the outer 'err'
//	    if err != nil { return err }
//	    err = ...() // This assigns to the inner 'err'
//	} else {
//	    err = ...() // Using the outer 'err' is fine here
//	}
//	return err  // Using outer 'err' after inner 'err' went out of scope
func (s UsageScope) Shadowing(selects SelectIndex, variable *types.Var) (*types.Var, token.Pos) {
	var (
		boundary        token.Pos       // Position where the shadowing ends
		crossedBoundary bool            // Inside switch, expand boundary
		selectBoundary  *ast.CommClause // Inside select, expand boundary
	)

	parent := variable.Parent() // The scope the variable declaration lives in
	switch node := s.Index[parent].(type) {
	case *ast.CaseClause:
		crossedBoundary = true

	case *ast.CommClause:
		selectBoundary = node

	case *ast.FuncType:
		return nil, token.NoPos // Variable declared at function top level - we don't cross them

	default:
		boundary = node.End()
	}

	// Search parent scopes for a variable with the same name and type
	for parent = parent.Parent(); parent != nil; parent = parent.Parent() {
		node := s.Index[parent]

		// Update boundary based on scope transitions
		switch {
		case crossedBoundary:
			// [ast.SwitchStmt] and [ast.TypeSwitchStmt] create an implicit scope.
			// Variables declared in a case clause are in the implicit scope of the case,
			// which is nested in the switch scope. We extend the boundary to the end of the switch scope.
			boundary, crossedBoundary = node.End(), false

		case selectBoundary != nil:
			// [ast.SelectStmt] does not create an implicit scope wrapping the cases.
			// We need extra bookkeeping at the caller to find the parent.
			boundary, selectBoundary = selects.Stmt(selectBoundary).End(), nil
		}

		exit := false

		// Adjust boundary for specific statement types
		switch node := node.(type) {
		case *ast.IfStmt:
			// Report shadowing after the outermost [ast.IfStmt] ends, even when reassigned in the else branch.
			boundary = node.End()

		case *ast.CaseClause:
			// Report shadowing after the switch. This allows usage inside the case without warnings.
			crossedBoundary = true

		case *ast.CommClause:
			selectBoundary = node

		case *ast.FuncType:
			// Don't cross function boundaries
			exit = true
		}

		// Look for a variable with the same name in this scope
		if shadowed := parent.Lookup(variable.Name()); shadowed != nil && shadowed.Pos() <= variable.Pos() {
			shadowedVar, ok := shadowed.(*types.Var)
			if !ok || !types.Identical(shadowedVar.Type(), variable.Type()) {
				// Not a variable, or has different type (e.g., x := x.(T) type assertion)
				return nil, token.NoPos
			}

			// Found a shadowed variable with matching name and type
			return shadowedVar, boundary
		}

		if exit {
			break
		}
	}

	return nil, token.NoPos
}
