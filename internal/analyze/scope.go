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
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"iter"
	"maps"
)

// ScopeAnalyzer provides scope analysis for variable movement.
//
// It maps scopes to their corresponding AST nodes and provides methods to:
//   - Determine if a variable can safely be moved to a tighter scope
//   - Find the tightest "safe" scope that avoids breaking semantics
type ScopeAnalyzer map[*types.Scope]ast.Node

// NewScopeAnalyzer creates a scope analyzer from the type checker's scope map.
func NewScopeAnalyzer(p Pass) ScopeAnalyzer {
	scopes := p.TypesInfo.Scopes

	s := make(ScopeAnalyzer, len(scopes))
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
func (s ScopeAnalyzer) Innermost(declScope *types.Scope, pos token.Pos) *types.Scope {
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

// Shadowed looks for a shadowed variable in parent scopes.
func (s ScopeAnalyzer) Shadowed(v *types.Var, pos token.Pos) types.Object {
	scope := v.Parent()
	if s.functionScope(scope) {
		return nil // Declared at the function top level
	}

	// Search in parent scopes
	for parent := scope.Parent(); parent != types.Universe; parent = parent.Parent() {
		if shadowed := parent.Lookup(v.Name()); shadowed != nil && shadowed.Pos() <= pos {
			if !types.Identical(shadowed.Type(), v.Type()) {
				return nil // Has different type, i.e. x := x.(T)
			}

			// Found a shadowed variable.
			return shadowed
		}

		if s.functionScope(parent) {
			return nil // Don't cross function definitions
		}
	}

	return nil
}

func (s ScopeAnalyzer) functionScope(scope *types.Scope) bool {
	_, ok := s[scope].(*ast.FuncType)
	return ok
}

// FindTargetNode finds a suitable node for moving a variable to a tighter scope.
func (s ScopeAnalyzer) FindTargetNode(declScope, targetScope *types.Scope, onlyBlock bool) (ast.Node, bool) {
	for scope := targetScope; scope != declScope; scope = scope.Parent() {
		targetNode, ok := s[scope]
		if !ok {
			panic("Invalid scope range")
		}

		var initField bool

		switch onlyBlock {
		case false:
			initField, ok = canUseNode(targetNode)

		case true:
			initField, ok = canUseBlockNode(targetNode)
		}

		if ok {
			return targetNode, initField
		}
	}

	return nil, false
}

// canUseNode determines if a variable can be moved to a given AST node.
//
// Returns:
//   - initField: true if the target is an Init field (if/for/switch statements)
//   - ok: true if the node can be used as a move target
func canUseNode(targetNode ast.Node) (initfield, ok bool) {
	switch n := targetNode.(type) {
	case *ast.IfStmt:
		return true, n.Init == nil

	case *ast.ForStmt:
		return true, n.Init == nil

	case *ast.SwitchStmt:
		return true, n.Init == nil

	case *ast.TypeSwitchStmt:
		return true, n.Init == nil

	case *ast.BlockStmt,
		*ast.CaseClause,
		*ast.CommClause:
		return false, true

	// case *ast.File, *ast.FuncType, *ast.TypeSpec, *ast.RangeStmt:
	default:
		return false, false
	}
}

// canUseBlockNode determines if a variable can be moved to a given AST node.
// This is a restricted version that only considers block scopes, not init fields.
//
// Returns:
//   - initField: false
//   - ok: true if the node can be used as a move target
func canUseBlockNode(targetNode ast.Node) (initfield, ok bool) {
	switch targetNode.(type) {
	case *ast.BlockStmt,
		*ast.CaseClause,
		*ast.CommClause:
		return false, true
	}

	return false, false
}

// FindSafeScope traverses from minScope up to declScope in the scope hierarchy,
// returning the tightest "safe" scope where a variable can be moved.
//
// "Safe" means the scope avoids moves that would change semantics:
//   - Loop bodies: Variables used in multiple iterations must stay outside the loop
//   - Function literals: Variables captured by closures must remain in the capturing scope
func (s ScopeAnalyzer) FindSafeScope(declScope, minScope *types.Scope) *types.Scope {
	var targetScope *types.Scope

	// The asymmetry between loops and functions requires a delayed update for FuncType:
	//   - Loop scopes (*ast.ForStmt): Contains Init/Cond/Post. The Body is in an *ast.BlockStmt.
	//   - Function scopes (*ast.FuncType): Contains parameters/result/body.
	crossedBoundary := true

	// Traverse upward through the scope chain (child → parent)
	for current := minScope; current != nil; {
		// Process delayed update from previous iteration
		if crossedBoundary {
			targetScope = current
			crossedBoundary = false
		}

		// Check the current scope for semantic boundaries
		switch s[current].(type) {
		case *ast.ForStmt:
			// Variables can safely move TO the loop scope (the Init field)
			// but cannot move INTO the loop body (would change lifetime semantics).
			// Immediate update: this scope is the boundary
			targetScope = current

		case *ast.RangeStmt:
			// Variables can stay in the loop scope (the Key field)
			// but cannot move INTO the loop body (would change lifetime semantics).
			// Immediate update: this scope is the boundary
			targetScope = current
			crossedBoundary = true

		case *ast.FuncType:
			// Variables CANNOT cross function literal boundaries because
			//  moving into the function would change closure capture semantics.
			crossedBoundary = true
		}

		// Check if we've reached the declaration scope
		if current == declScope {
			return targetScope
		}

		// Calculate parent, skip case scopes when the current is not in the body
		parent := current.Parent()
		switch n := s[parent].(type) {
		case *ast.CaseClause:
			if current.Pos() < n.Colon {
				parent = parent.Parent()
			}

		case *ast.CommClause:
			if current.Pos() < n.Colon {
				parent = parent.Parent()
			}
		}

		current = parent
	}

	// This should never happen in normal operation - it would mean declScope
	// is not an ancestor of minScope, which violates our preconditions
	return nil
}

// CommonAncestor finds the lowest common ancestor (LCA) of two scopes in the scope tree.
//
//   - declScope: The declaration scope (root of the subtree we're searching)
//   - currentScope: First scope (the current minimum scope)
//   - usageScope: Second scope (scope of the new use we're processing)
func (s ScopeAnalyzer) CommonAncestor(declScope, currentScope, usageScope *types.Scope) *types.Scope {
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
func (s ScopeAnalyzer) parentScopes(root, start *types.Scope) iter.Seq2[*types.Scope, struct{}] {
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
func (s ScopeAnalyzer) parentScope(scope *types.Scope) *types.Scope {
	parent := scope.Parent()
	switch n := s[parent].(type) {
	case *ast.CaseClause:
		if n.Colon > scope.Pos() {
			parent = parent.Parent()
		}

	case *ast.CommClause:
		if n.Colon > scope.Pos() {
			parent = parent.Parent()
		}
	}

	return parent
}

// scopeName returns a human-readable name for the scope type.
func scopeName(node ast.Node) string {
	switch node.(type) {
	// keep-sorted start newline_separated=yes
	case *ast.BlockStmt:
		return "block"

	case *ast.CaseClause:
		return "case"

	case *ast.CommClause:
		return "select case"

	case *ast.File:
		return "file"

	case *ast.ForStmt:
		return "for"

	case *ast.FuncType:
		return "function"

	case *ast.IfStmt:
		return "if"

	case *ast.RangeStmt:
		return "range"

	case *ast.SwitchStmt:
		return "switch"

	case *ast.TypeSwitchStmt:
		return "type switch"

	case nil:
		return "<nil>"

	default:
		return fmt.Sprintf("nested: %T", node)
		// keep-sorted end
	}
}
