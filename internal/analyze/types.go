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

const invalidNode nodeIndex = -1

// scopeRange represents the scope range for a declaration.
type scopeRange = struct {
	decl, // The scope where the variable was declared
	usage *types.Scope // The tightest scope containing all uses (LCA of all usage scopes)
}

// usageResult contains the scope analysis for all variable declarations from stage 1.
type usageResult = struct {
	// Map from declaration indices to their computed scope ranges.
	// The scopeRange.usage field represents the tightest scope containing all uses.
	scopeRanges map[nodeIndex]scopeRange

	// Map of variables to usage.
	usages map[*types.Var][]nodeUsage
}

// nodeUsage tracks a single usage of a declaration.
type nodeUsage = struct {
	decl nodeIndex
	used bool
}

// moveTarget represents a declaration that can be moved to a tighter scope.
type moveTarget = struct {
	targetNode ast.Node  // The node with  the target scope (e.g., *ast.IfStmt, *ast.BlockStmt)
	unused     []string  // Unused identifiers in this declaration
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
