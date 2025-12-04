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
	"go/token"
	"go/types"
	"iter"
)

// scopeAnalyzer provides scope analysis for variable movement.
//
// It maps scopes to their corresponding AST nodes and provides methods to:
//   - Determine if a variable can safely be moved to a tighter scope
//   - Find the tightest "safe" scope that avoids breaking semantics
type scopeAnalyzer map[*types.Scope]ast.Node

// scopeAnalyzer creates a scope analyzer from the type checker's scope map.
func (p pass) scopeAnalyzer() scopeAnalyzer {
	scopes := p.TypesInfo.Scopes

	s := make(scopeAnalyzer, len(scopes))
	for node, scope := range scopes {
		s[scope] = node
	}

	return s
}

// innermost finds the innermost scope containing a use, with special handling
// for case/select expressions.
//
// For most positions, this returns the innermost scope from the type checker. However,
// when a variable is used in a case or select expression (between "case" and ":" tokens),
// it adjusts the scope to the parent.
func (s scopeAnalyzer) innermost(declScope *types.Scope, pos token.Pos) *types.Scope {
	usageScope := declScope.Innermost(pos)
	switch usageScope {
	case declScope, nil:
		return usageScope
	}

	// Special handling: if the variable is used in case/select expression,
	// adjust scope to parent to prevent moving a declaration into the case body
	switch node := s[usageScope].(type) {
	case *ast.CaseClause:
		if contains(node.Case, node.Colon, pos) {
			// The variable is used in the case expression: case x == 0:
			// Can't move x into this case's scope
			usageScope = usageScope.Parent()
		}

	case *ast.CommClause:
		if contains(node.Case, node.Colon, pos) {
			// The variable is used in the send/receive: case ch <- x:
			// Can't move x into this select case's scope
			usageScope = usageScope.Parent()
		}
	}

	return usageScope
}

// findTargetNode determines if a short variable declaration can be moved to a tighter scope.
func (s scopeAnalyzer) findTargetNode(declScope, targetScope *types.Scope) (ast.Node, bool) {
	// Verify the result is actually tighter than the current declaration
	for scope := targetScope; scope != declScope; scope = scope.Parent() {
		// Look up the AST node for the safe minimum scope
		targetNode, ok := s[scope]
		if !ok { // This should always succeed, but we check to be defensive
			return nil, false
		}

		if initField, ok := canUseNode(targetNode); ok {
			return targetNode, initField
		}
	}

	return nil, false
}

// findTargetNodeInBlock finds a suitable block-scope node for moving a variable.
//
// This is a restricted version of findTargetNode that only considers
// block scopes (BlockStmt, CaseClause, CommClause).
func (s scopeAnalyzer) findTargetNodeInBlock(declScope, targetScope *types.Scope) ast.Node {
	// Verify the result is actually tighter than the current declaration
	for scope := targetScope; scope != declScope; scope = scope.Parent() {
		// Look up the AST node for the safe minimum scope
		targetNode, ok := s[scope]
		if !ok { // This should always succeed, but we check to be defensive
			return nil
		}

		switch targetNode.(type) {
		case *ast.BlockStmt,
			*ast.CaseClause,
			*ast.CommClause:
			return targetNode
		}
	}

	return nil
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

	case *ast.RangeStmt:
		return false, false

	case *ast.SwitchStmt:
		return true, n.Init == nil

	case *ast.TypeSwitchStmt:
		return true, n.Init == nil

	case *ast.BlockStmt,
		*ast.CaseClause,
		*ast.CommClause:
		return false, true

	// case *ast.File, *ast.FuncType, *ast.TypeSpec:
	default:
		return false, false
	}
}

// findSafeScope traverses from minScope up to declScope in the scope hierarchy,
// returning the tightest "safe" scope where a variable can be moved.
//
// "Safe" means the scope avoids moves that would change semantics:
//   - Loop bodies: Variables used in multiple iterations must stay outside the loop
//   - Function literals: Variables captured by closures must remain in the capturing scope
func (s scopeAnalyzer) findSafeScope(declScope, minScope *types.Scope) *types.Scope {
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
		case nil:
			break

		case *ast.CaseClause:
			if n.Colon > current.Pos() {
				parent = parent.Parent()
			}

		case *ast.CommClause:
			if n.Colon > current.Pos() {
				parent = parent.Parent()
			}
		}

		current = parent
	}

	// This should never happen in normal operation - it would mean declScope
	// is not an ancestor of minScope, which violates our preconditions
	return nil
}

// contains checks if a token position falls within a range.
func contains(left, right, pos token.Pos) bool {
	return left <= pos && pos <= right
}

// commonAncestor finds the lowest common ancestor (LCA) of two scopes in the scope tree.
//
//   - declScope: The declaration scope (root of the subtree we're searching)
//   - currentScope: First scope (the current minimum scope)
//   - usageScope: Second scope (scope of the new use we're processing)
func (s scopeAnalyzer) commonAncestor(declScope, currentScope, usageScope *types.Scope) *types.Scope {
	switch usageScope {
	case currentScope, // Same scope as before: no change needed
		declScope: // Tightest possible
		return usageScope
	}

	// Phase 1: Build a path from currentScope to declScope
	// This creates a set of all scopes in the path
	path := make(map[*types.Scope]struct{})
	for scope := range s.parentScopes(currentScope, declScope) {
		path[scope] = struct{}{}
	}

	// Phase 2: Walk from usageScope to declScope
	// Return the first scope that exists in both paths (the LCA)
	for scope := range s.parentScopes(usageScope, declScope) {
		if _, ok := path[scope]; ok {
			return scope
		}
	}

	// If we reach here, LCA is the declScope itself
	// This means the two scopes are in completely different branches
	return declScope
}

// parentScopes yields a sequence of scopes from start up to (but not including) root.
func (s scopeAnalyzer) parentScopes(start, root *types.Scope) iter.Seq[*types.Scope] {
	return func(yield func(*types.Scope) bool) {
		for scope := start; scope != root; {
			if !yield(scope) {
				break
			}

			// Calculate parent, skip case scopes when the current is not in the body
			parent := scope.Parent()
			switch n := s[parent].(type) {
			case nil:
				break

			case *ast.CaseClause:
				if n.Colon > scope.Pos() {
					parent = parent.Parent()
				}

			case *ast.CommClause:
				if n.Colon > scope.Pos() {
					parent = parent.Parent()
				}
			}

			scope = parent
		}
	}
}
