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
	"maps"

	"golang.org/x/tools/go/ast/astutil"
)

// ScopeAnalyzer provides scope analysis for variable movement.
//
// It maps scopes to their corresponding AST nodes and provides methods to:
//   - Determine if a variable can safely be moved to a tighter scope
//   - Find the tightest "safe" scope that avoids breaking semantics
type ScopeAnalyzer map[*types.Scope]ast.Node

// NewScopeAnalyzer creates a scope analyzer from the type checker's scope map.
func NewScopeAnalyzer(scopes map[ast.Node]*types.Scope) ScopeAnalyzer {
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

// TargetNode finds a suitable node for moving a variable to a tighter scope.
//
// Parameters:
//   - declScope: The scope where the variable is currently declared
//   - targetScope: The tightest scope containing all variable uses
//   - maxPos: Position we should not cross that blocks the move
//   - onlyBlock: If true, only consider block scopes (not init fields)
func (s ScopeAnalyzer) TargetNode(declScope, targetScope *types.Scope, maxPos token.Pos, onlyBlock bool) ast.Node {
	// Walk up from targetScope toward declScope, returning the first suitable node.
	for scope := targetScope; scope != declScope; scope = scope.Parent() {
		//  If maxPos is set, scopes starting after it are skipped.
		if maxPos.IsValid() && scope.Pos() > maxPos {
			continue
		}

		targetNode, ok := s[scope]
		if !ok {
			panic("Invalid scope range")
		}

		switch onlyBlock {
		case false:
			if canUseNode(targetNode) {
				return targetNode
			}

		case true:
			if canUseBlockNode(targetNode) {
				return targetNode
			}
		}
	}

	return nil
}

// canUseNode determines if a variable can be moved to a given AST node.
func canUseNode(targetNode ast.Node) bool {
	switch n := targetNode.(type) {
	case *ast.IfStmt:
		return n.Init == nil

	case *ast.ForStmt:
		return n.Init == nil

	case *ast.SwitchStmt:
		return n.Init == nil

	case *ast.TypeSwitchStmt:
		return n.Init == nil

	case *ast.BlockStmt,
		*ast.CaseClause,
		*ast.CommClause:
		return true

	// case *ast.File, *ast.FuncType, *ast.TypeSpec, *ast.RangeStmt:
	default:
		return false
	}
}

// canUseBlockNode determines if a variable can be moved to a given AST node.
// This is a restricted version that only considers block scopes, not init fields.
func canUseBlockNode(targetNode ast.Node) bool {
	switch targetNode.(type) {
	case *ast.BlockStmt,
		*ast.CaseClause,
		*ast.CommClause:
		return true

	default:
		return false
	}
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

		// Calculate parent
		current = s.parentScope(current)
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

	// Skip case scopes when the current scope is not in the body.
	// Note: The parent of *ast.CaseClause expressions is the switch expression
	if n, ok := s[parent].(*ast.CommClause); ok && scope.Pos() < n.Colon {
		parent = parent.Parent()
	}

	return parent
}

// ScopeName returns a human-readable name for the scope type.
func ScopeName(node ast.Node) string {
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
		return astutil.NodeDescription(node)
		// keep-sorted end
	}
}
