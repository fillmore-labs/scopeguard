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

// NodeIndex is the index from [inspector.Cursor], increases monotonically throughout the traversal.
type NodeIndex = int32

// InvalidNode represents an invalid node index.
const InvalidNode NodeIndex = -1

// ScopeRange represents the scope range for a declaration.
type ScopeRange struct {
	// Decl is the scope where the variable was declared
	Decl,
	// Usage is the tightest scope containing all uses
	Usage *types.Scope
}

// UsageResult contains the scope analysis for all variable declarations from stage 1.
type UsageResult struct {
	// Map from declaration indices to their computed scope ranges.
	ScopeRanges map[NodeIndex]ScopeRange

	// Map of variables to usage.
	Usages map[*types.Var][]NodeUsage
}

// ShadowUse contains information about a variable use after previously shadowed.
type ShadowUse struct {
	Var       *types.Var
	Use, Decl NodeIndex
}

// ShadowUsed contains information about variables use after previously shadowed.
type ShadowUsed []ShadowUse

// NestedAssign contains information about a nested variable assign.
type NestedAssign struct {
	id   *ast.Ident
	stmt ast.Node
}

// NestedAssigned contains information about nested variable assigns.
type NestedAssigned []NestedAssign

// NodeUsage tracks a single usage of a declaration.
type NodeUsage struct {
	Decl  NodeIndex
	Flags UsageFlags
}

// UsageFlags indicates how a variable is used.
type UsageFlags uint8

const (
	// UsageUsed indicates the variable declaration is used.
	UsageUsed UsageFlags = 1 << iota
	// UsageTypeChange indicates the variable redeclaration implies a type change.
	UsageTypeChange
	// UsageUntypedNil indicates the variable redeclaration is assigned an untyped nil.
	UsageUntypedNil

	// UsageNone indicates the variable declaration is unused.
	UsageNone UsageFlags = 0
)

// MoveTarget represents a declaration that can be moved to a tighter scope.
type MoveTarget struct {
	TargetNode ast.Node   // The node with the target scope (e.g., *[ast.IfStmt], *[ast.BlockStmt])
	Unused     []string   // Unused identifiers in this declaration
	Decl       NodeIndex  // Inspector index of the declaration statement to move
	Status     MoveStatus // Status indicating if the move is safe or why it isn't
}

// TargetResult is the complete set of declarations that can be moved to tighter scopes.
//
// This slice is sorted by source position to ensure
// deterministic diagnostic ordering in the analyzer output.
type TargetResult struct {
	Move []MoveTarget
}
