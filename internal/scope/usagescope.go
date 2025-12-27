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
func (s UsageScope) Shadowing(v *types.Var, pos token.Pos) (*types.Var, token.Pos) {
	scope := v.Parent()
	start := scope.End()

	switch s.Index[scope].(type) {
	case *ast.FuncType:
		return nil, token.NoPos // Declared at the function top level

	case *ast.CaseClause:
		start = scope.Parent().End() // Complain after the switch

	case *ast.CommClause:
		// The parent of an *ast.CommClause scope is the parent of the select statement.
	}

	// Search in parent scopes
	for parent := scope.Parent(); parent != nil; parent = parent.Parent() {
		if shadowed := parent.Lookup(v.Name()); shadowed != nil && shadowed.Pos() <= pos {
			shadowed, ok := shadowed.(*types.Var)
			if !ok || !types.Identical(shadowed.Type(), v.Type()) {
				return nil, token.NoPos // Has different type, i.e. x := x.(T)
			}

			// Found a shadowed variable.
			return shadowed, start
		}

		switch s.Index[parent].(type) {
		case *ast.FuncType:
			return nil, token.NoPos // Don't cross function definitions

		case *ast.CaseClause:
			// Complain after the switch. This makes shadowing inside the case possible without notifications.
			start = parent.Parent().End()
		}
	}

	return nil, token.NoPos
}
