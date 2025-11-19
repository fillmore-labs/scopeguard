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
)

// scopeAnalyzer provides scope analysis for variable movement.
//
// It maps scopes to their corresponding AST nodes and provides methods to:
//   - Determine if a variable can safely be moved to a tighter scope
//   - Find the tightest "safe" scope that avoids breaking semantics
//   - Check scope characteristics (loops, function literals, etc.)
//
// The analyzer is bidirectional: given a scope, find its AST node, and
// given the scope's parent chain, traverse upward through the AST structure.
type scopeAnalyzer map[*types.Scope]ast.Node

// scopeAnalyzer creates a scope analyzer from the type checker's scope map.
//
// The type checker's Scopes map is indexed by AST node (node → scope).
// We invert it to create a lookup from scope → node, which allows us to:
//  1. Find the AST node associated with a scope
//  2. Classify scopes by their node type (loop, function, block, etc.)
//  3. Traverse both the scope chain (via Parent()) and AST structure simultaneously
func (p pass) scopeAnalyzer() scopeAnalyzer {
	scopes := p.pass.TypesInfo.Scopes

	s := make(scopeAnalyzer, len(scopes))
	for node, scope := range scopes {
		s[scope] = node
	}

	return s
}

// innermostScope finds the innermost scope containing a use, with special handling
// for case/select expressions.
//
// For most positions, this returns the innermost scope from the type checker. However,
// when a variable is used in a case or select expression (between "case" and ":" tokens),
// it adjusts the scope to the parent to prevent moving declarations into case bodies.
//
// Example:
//
//	switch x {
//	case foo(y):  // y used in case expression
//	    ...
//	}
//
// If we moved y into the case clause's scope, it would only be accessible inside that
// specific case, not in the expression that selects the case.
//
// Returns:
//   - *types.Scope: The adjusted innermost scope
//   - bool: true if successful, false if scope lookup failed
func (s scopeAnalyzer) innermostScope(declScope *types.Scope, pos token.Pos) (*types.Scope, bool) {
	usageScope := declScope.Innermost(pos)
	switch usageScope {
	case declScope, nil:
		return usageScope, true
	}

	// Special handling: if the variable is used in case/select expression,
	// adjust scope to parent to prevent moving declaration into the case body
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

	return usageScope, true
}

// findTargetNode determines if a short variable declaration can be moved to a tighter scope.
//
// It performs two checks:
//  1. Is there a tighter scope to move to? (targetScope != declScope)
//  2. Can we use the AST node of the target scope?
//
// Returns:
//   - bool: true if move should be to an init field (only one possible)
//   - ast.Node: The AST node representing the target scope
func (s scopeAnalyzer) findTargetNode(declScope, targetScope *types.Scope) (bool, ast.Node) {
	// Verify the result is actually tighter than the current declaration
	for scope := targetScope; scope != declScope; scope = scope.Parent() {
		// Look up the AST node for the safe minimum scope
		targetNode, ok := s[scope]
		if !ok { // This should always succeed, but we check to be defensive
			return false, nil
		}

		if initField, ok := canUseNode(targetNode); ok {
			return initField, targetNode
		}
	}

	return false, nil
}

// findTargetNodeInBlock finds a suitable block-scope AST node for moving a variable.
//
// This is a restricted version of findTargetNode that only considers
// block scopes (BlockStmt, CaseClause, CommClause). It's used for var declarations
// since Go syntax doesn't allow "var" in Init fields.
//
// Returns:
//   - ast.Node: The target node, or nil if no suitable node is found
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
// Returns two values:
//  1. initField (bool): true if the target is an Init field (if/for/switch statements)
//  2. ok (bool): true if the node can be used as a move target
//
// Supported target nodes:
//   - *ast.IfStmt, *ast.ForStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
//     Can use if Init field is nil (returns initField=true, ok=true)
//   - *ast.BlockStmt, *ast.CaseClause, *ast.CommClause:
//     Always usable as block scopes (returns initField=false, ok=true)
//   - *ast.RangeStmt:
//     Cannot use (no Init field available; returns initField=false, ok=false)
//
// Returns (false, false) for unsupported node types (e.g., *ast.File, *ast.FuncType).
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
	for current := minScope; current != nil; current = current.Parent() {
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
	}

	// This should never happen in normal operation - it would mean declScope
	// is not an ancestor of minScope, which violates our preconditions
	return nil
}

// contains checks if a token position falls within a range.
// Used to detect if a variable use is within a case/select expression.
func contains(left, right, pos token.Pos) bool {
	return left <= pos && pos <= right
}
