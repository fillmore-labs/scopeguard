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
)

// TargetScope determines where declarations can be safely moved.
// It extends ScopeIndex with target-specific scope safety analysis.
type TargetScope struct {
	Index
}

// NewTargetScope creates a new [TargetScope] instance.
func NewTargetScope(scopes Index) TargetScope {
	return TargetScope{Index: scopes}
}

// FindSafeScope traverses from minScope up to declScope in the scope hierarchy,
// returning the tightest "safe" scope where a variable can be moved.
//
// "Safe" means the scope avoids moves that would change semantics:
//   - Loop bodies: Variables used in multiple iterations must stay outside the loop
//   - Function literals: Variables captured by closures must remain in the capturing scope
func (s TargetScope) FindSafeScope(declScope, minScope *types.Scope) *types.Scope {
	// The asymmetry between loops and functions requires a delayed update for FuncType:
	//   - Loop scopes (*ast.ForStmt): Contains Init/Cond/Post. The Body is in an *ast.BlockStmt.
	//   - Function scopes (*ast.FuncType): Contains parameters/result/body.
	targetScope, crossedBoundary := minScope, false

	// Traverse upward through the scope chain (child → parent)
	for current := minScope; current != nil; current = s.ParentScope(current) {
		// Process delayed update from previous iteration
		if crossedBoundary {
			targetScope, crossedBoundary = current, false
		}

		// Check the current scope for semantic boundaries
		switch s.Index[current].(type) {
		case *ast.ForStmt:
			// Variables can safely move TO the loop scope (the Init field)
			// but cannot move INTO the loop body (would change lifetime semantics).
			// Immediate update: this scope is the boundary
			targetScope = current

		case *ast.RangeStmt:
			// Variables can stay in the loop scope (the Key field)
			// but cannot move INTO the loop body (would change lifetime semantics).
			// Immediate update: this scope is the boundary
			targetScope, crossedBoundary = current, true

		case *ast.FuncType:
			// Variables CANNOT cross function literal boundaries because
			//  moving into the function would change closure capture semantics.
			crossedBoundary = true
		}

		if current == declScope {
			// We've reached the declaration scope
			return targetScope
		}
	}

	// This should never happen in normal operation - it would mean declScope
	// is not an ancestor of minScope, which violates our preconditions
	return nil
}

// TargetNode finds a suitable node for moving a variable to a tighter scope.
//
// Parameters:
//   - declScope: The scope where the variable is currently declared
//   - targetScope: The tightest scope containing all variable uses
//   - maxPos: Position we should not cross that blocks the move
//   - onlyBlock: If true, only consider block scopes (not init fields)
func (s TargetScope) TargetNode(declScope, targetScope *types.Scope, maxPos token.Pos, onlyBlock bool) ast.Node {
	// Walk up from targetScope toward declScope, returning the first suitable node.
	for scope := targetScope; scope != declScope; scope = scope.Parent() {
		//  If maxPos is set, scopes starting after it are skipped.
		if maxPos.IsValid() && scope.Pos() > maxPos {
			continue
		}

		targetNode, ok := s.Index[scope]
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
