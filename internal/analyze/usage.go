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
	"context"
	"go/ast"
	"go/types"
	"runtime/trace"

	"golang.org/x/tools/go/ast/inspector"
)

// usage tracks variable uses and computes the minimum safe scope for each declaration.
//
// This is Stage 2 of the analyzer pipeline. For each variable use, it:
//   - Finds the innermost scope containing that use
//   - Computes the lowest common ancestor (LCA) with previous uses
func (p pass) usage(ctx context.Context, in *inspector.Inspector, scopes scopeAnalyzer, decls declResult) (usageResult, error) {
	defer trace.StartRegion(ctx, "usage").End()

	result := usageResult{
		scopes: make(map[nodeIndex]scopeRange),
	}

	for id, obj := range p.pass.TypesInfo.Uses {
		v, ok := obj.(*types.Var)
		if !ok {
			continue // Not a local variable
		}

		declIndices, ok := decls.decls[v]
		if !ok {
			continue // Variable isn't tracked (e.g., named return, function param, package-level)
		}

		// Find the most recent declaration before this use.
		// This handles variables with multiple declarations (redeclarations).
		declIndex, ok := findDeclaration(in, id, declIndices)
		if !ok {
			continue // Redeclaration or declaration not recorded
		}

		// Filter out redeclarations of named return values.
		// These cannot be moved because bare return statements can use
		// the named return anywhere in the function body.
		if _, notMovable := decls.notMovable[declIndex]; notMovable {
			continue
		}

		declScope := v.Parent()

		currentScope, hasScope := result.scopes[declIndex]

		if hasScope && currentScope.usage == declScope {
			continue // Already at the innermost scope (can't move tighter)
		}

		// Find the innermost scope containing this use
		usageScope, ok := scopes.innermostScope(declScope, id.NamePos)
		if !ok {
			continue
		}

		var (
			common   *types.Scope
			newScope bool
		)

		if !hasScope {
			// First use: set target scope
			result.scopes[declIndex] = scopeRange{declScope, usageScope}
			continue
		}

		// Compute the minimum scope that contains all uses so far
		if common, newScope = commonAncestor(declScope, currentScope.usage, usageScope); newScope {
			result.scopes[declIndex] = scopeRange{declScope, common}
		}
	}

	// Identify unused declarations and mark those that must be kept for type safety
	// The returned map's boolean value indicates: true = keep, false = can remove
	if unused := p.findUnused(in, decls, result.scopes); len(unused) > 0 {
		result.unused = unused
	}

	return result, nil
}

// findDeclaration identifies which declaration owns a particular variable use.
//
// For variables with redeclarations, this function:
//  1. Searches backwards through declIndices to find the most recent declaration
//  2. Handles the case where a variable is used in its own redeclaration
//
// # Parameters
//   - in: Inspector for accessing AST nodes by index
//   - id: The identifier representing the variable use
//   - declIndices: All declaration indices for this variable (by traversal order)
//
// # Returns
//   - nodeIndex: The inspector index that owns this use
//   - bool: true when found
func findDeclaration(in *inspector.Inspector, id *ast.Ident, declIndices []nodeIndex) (index nodeIndex, ok bool) {
	for j := len(declIndices) - 1; j >= 0; j-- {
		index = declIndices[j]

		if node := in.At(index).Node(); node.Pos() <= id.NamePos {
			// Found a declaration that begins before this identifier
			switch stmt, isAssign := node.(*ast.AssignStmt); {
			case !isAssign, // Not a short variable declaration (e.g., var statement)
				id.NamePos > stmt.End(): // Use occurs after the declaration
				return index, true

			case id.NamePos < stmt.TokPos, // Identifier appears before := (redeclaration, doesn't count as use)
				j == 0: // No previous definition recorded (e.g., function parameter)
				return -1, false

			default: // Identifier used in RHS of this declaration, refers to previous declaration
				return declIndices[j-1], true
			}
		}
	}

	return -1, false // Definition not recorded
}

// commonAncestor finds the lowest common ancestor (LCA) of two scopes in the scope tree.
//
// Parameters:
//   - declScope: The declaration scope (root of the subtree we're searching)
//   - infoScope: First scope (the current minimum scope)
//   - usageScope: Second scope (scope of the new use we're processing)
//
// Returns:
//   - *types.Scope: The tightest scope that contains both infoScope and usageScope
func commonAncestor(declScope, currentScope, usageScope *types.Scope) (*types.Scope, bool) {
	switch usageScope {
	case currentScope:
		// Same scope as before: no change needed
		return usageScope, false

	case declScope:
		// Tightest possible: set target scope
		return usageScope, true
	}

	// Phase 1: Build a path from infoScope to declScope
	// This creates a set of all scopes in the path
	path := make(map[*types.Scope]struct{})
	for scope := currentScope; scope != declScope; scope = scope.Parent() {
		path[scope] = struct{}{}
	}

	// Phase 2: Walk from usageScope to declScope
	// Return the first scope that exists in both paths (the LCA)
	for scope := usageScope; scope != declScope; scope = scope.Parent() {
		if _, ok := path[scope]; ok {
			return scope, true
		}
	}

	// If we reach here, LCA is the declScope itself
	// This means the two scopes are in completely different branches
	return declScope, true
}
