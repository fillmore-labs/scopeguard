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
	"go/types"
)

// nodeIndex is the index from [inspector.Cursor], increases monotonically throughout the traversal.
type nodeIndex = int32

// declResult contains all variable declarations collected in Stage 1.
//
// Variables can have multiple declaration indices when they're redeclared
// (e.g., x := 1; later: x, y := foo()). The indices are ordered by source position.
//
// Named return values receive special handling:
//   - They're NOT included in decls (they're part of function signature, not movable)
//   - Their redeclarations ARE marked in notMovable (so Stage 2 can filter them out)
type declResult = struct {
	// Map of variables to their declaration node indices.
	// Includes both initial declarations and redeclarations (except for named returns).
	decls map[*types.Var][]nodeIndex

	// Indices of short declarations that redefine named return values.
	// These cannot be moved because bare return statements can use named returns
	// anywhere in the function body.
	notMovable map[nodeIndex]struct{}
}

// scopeRange represents the scope range for a variable declaration.
//
// It tracks both where a variable is declared and the tightest scope
// that contains all its uses, which is the minimum scope the declaration can be moved to.
type scopeRange = struct {
	decl, // The scope where the variable was declared
	usage *types.Scope // The tightest scope containing all uses (LCA of all usage scopes)
}

// usageResult contains the scope analysis for all variable declarations from Stage 2.
//
// This is the output of the usage phase, which tracks variable uses and computes
// the minimum scope for each declaration by finding the lowest common ancestor
// of all usage scopes.
type usageResult = struct {
	// Map from declaration indices to their computed scope ranges.
	// The scopeRange.usage field represents the tightest scope containing all uses.
	scopes map[nodeIndex]scopeRange

	// Map of unused variable declarations.
	// The boolean value indicates: true = keep for type safety, false = can be removed.
	unused map[nodeIndex]bool
}

// moveTarget represents a declaration that can be moved to a tighter scope.
//
// This is the output of the target phase, containing all information needed to:
//  1. Locate the declaration in the AST (via decl index)
//  2. Determine whether to generate a fix (via flag dontFix)
//  3. Generate the diagnostic message and fix (with targetNode as target)
type moveTarget = struct {
	targetNode ast.Node  // The AST node representing the target scope (e.g., *ast.IfStmt, *ast.BlockStmt)
	decl       nodeIndex // Inspector index of the declaration statement to move
	dontFix    bool      // Don't apply fixes
}

// targetResult is the complete set of declarations that can be moved to tighter scopes.
//
// This slice is sorted by source position to ensure
// deterministic diagnostic ordering in the analyzer output.
type targetResult = struct {
	move []moveTarget
}
