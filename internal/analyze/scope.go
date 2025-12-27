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

package analyze

import (
	"go/ast"
	"go/token"
	"go/types"
	"iter"
	"maps"
)

// ScopeIndex provides scope analysis for variable movement.
//
// It maps scopes to their corresponding AST nodes and provides methods to:
//   - Determine if a variable can safely be moved to a tighter scope
//   - Find the tightest "safe" scope that avoids breaking semantics
type ScopeIndex map[*types.Scope]ast.Node

// NewScopeIndex creates a scope analyzer from the type checker's scope map.
func NewScopeIndex(scopes map[ast.Node]*types.Scope) ScopeIndex {
	s := make(ScopeIndex, len(scopes))
	for node, scope := range scopes {
		s[scope] = node
	}

	return s
}

// Innermost finds the innermost scope containing a use, with special handling
// for case/select expressions.
//
// For most positions, this returns the innermost scope from the type checker. However,
// when a variable is used in a case or select expression (between "case" and ":" tokens),
// it adjusts the scope to the parent.
func (s ScopeIndex) Innermost(declScope *types.Scope, pos token.Pos) *types.Scope {
	usageScope := declScope.Innermost(pos)
	switch usageScope {
	case declScope, nil:
		return usageScope
	}

	// Special handling: if the variable is used in case/select expression,
	// adjust scope to parent to prevent moving a declaration into the case body
	switch n := s[usageScope].(type) {
	case *ast.CaseClause:
		if pos < n.Colon {
			// The variable is used in the case expression: case x == 0:
			// Can't move x into this case's scope
			usageScope = usageScope.Parent()
		}

	case *ast.CommClause:
		if pos < n.Colon {
			// The variable is used in the send/receive: case ch <- x:
			// Can't move x into this select case's scope
			usageScope = usageScope.Parent()
		}
	}

	return usageScope
}

// Shadowing looks for a shadowed variable in parent scopes.
func (s ScopeIndex) Shadowing(v *types.Var, pos token.Pos) (*types.Var, token.Pos) {
	scope := v.Parent()
	start := scope.End()

	switch s[scope].(type) {
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

		switch s[parent].(type) {
		case *ast.FuncType:
			return nil, token.NoPos // Don't cross function definitions

		case *ast.CaseClause:
			// Complain after the switch. This makes shadowing inside the case possible without notifications.
			start = parent.Parent().End()
		}
	}

	return nil, token.NoPos
}

// CommonAncestor finds the lowest common ancestor (LCA) of two scopes in the scope tree.
//
//   - declScope: The declaration scope (root of the subtree we're searching)
//   - currentScope: First scope (the current minimum scope)
//   - usageScope: Second scope (scope of the new use we're processing)
func (s ScopeIndex) CommonAncestor(declScope, currentScope, usageScope *types.Scope) *types.Scope {
	switch usageScope {
	case currentScope, // Same scope as before: no change needed
		declScope: // Tightest possible
		return usageScope
	}

	// Phase 1: Build a path from currentScope to declScope
	// This creates a set of all scopes in the path
	path := maps.Collect(s.parentScopes(declScope, currentScope))

	// Phase 2: Walk from usageScope to declScope
	// Return the first scope that exists in both paths (the LCA)
	for scope := range s.parentScopes(declScope, usageScope) {
		if _, ok := path[scope]; ok {
			return scope
		}
	}

	// If we reach here, LCA is the declScope itself
	// This means the two scopes are in completely different branches
	return declScope
}

// parentScopes yields a sequence of scopes from start up to (but not including) root.
func (s ScopeIndex) parentScopes(root, start *types.Scope) iter.Seq2[*types.Scope, struct{}] {
	return func(yield func(*types.Scope, struct{}) bool) {
		for scope := start; scope != root; scope = s.parentScope(scope) {
			if scope == nil { // Reached the [types.Universe] scope
				panic("start scope is not in root")
			}

			if !yield(scope, struct{}{}) {
				break
			}
		}
	}
}

// Calculate parent, skip case scopes when the current scope is not in the body but the expression.
func (s ScopeIndex) parentScope(scope *types.Scope) *types.Scope {
	parent := scope.Parent()

	// Skip case scopes when the current scope is not in the body.
	// Note: The parent of *ast.CaseClause expressions is the switch expression
	if n, ok := s[parent].(*ast.CommClause); ok && scope.Pos() < n.Colon {
		parent = parent.Parent()
	}

	return parent
}
