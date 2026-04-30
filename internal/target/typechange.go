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

package target

import (
	"go/types"
	"iter"

	"fillmore-labs.com/scopeguard/internal/astutil"
	"fillmore-labs.com/scopeguard/internal/category"
	"fillmore-labs.com/scopeguard/internal/usage"
)

// BlockMovesWithTypeChanges marks candidates as blocked when moving would change
// the inferred type of a variable that is actually used.
func (cm CandidateManager) BlockMovesWithTypeChanges(allDeclarations iter.Seq2[*types.Var, []usage.DeclarationNode]) {
	for _, declarations := range allDeclarations {
		for _, declaration := range declarations {
			if !declaration.Usage.Has(usage.UsageUsedAndTypeChange) {
				continue
			}

			m, ok := cm.candidates[declaration.Decl]
			if !ok || !m.movable() {
				continue
			}

			m.Status = category.MoveBlockedTypeChange
			cm.candidates[declaration.Decl] = m
		}
	}
}

// BlockMovesLosingTypeInfo prevents moves that would lose necessary type information.
//
// Scenario: A variable is declared with an explicit or inferred type, then later reassigned
// with a different type inference. If we move the first declaration, subsequent uses would
// have a different type.
//
// Example:
//
//	var x any           // First declaration (unused)
//	x, y := "hello", 0  // Reassignment with different type
//
// Moving the first declaration would change x's type from any to string.
func (cm CandidateManager) BlockMovesLosingTypeInfo(allDeclarations iter.Seq2[*types.Var, []usage.DeclarationNode]) map[astutil.NodeIndex][]*types.Var {
	unused := make(map[astutil.NodeIndex][]*types.Var)

	for v, declarations := range allDeclarations {
		// If type info preservation is needed, the first declaration is effectively used (for type info)
		keepTypeInfo := cm.evaluateTypeConstraints(declarations)

		for _, declaration := range declarations {
			if keepTypeInfo {
				keepTypeInfo = false
				continue
			}

			// Populate unused map
			if !declaration.Usage.Has(usage.UsageUsed) {
				unused[declaration.Decl] = append(unused[declaration.Decl], v)
			}
		}
	}

	return unused
}

// evaluateTypeConstraints checks if valid type constraints exist that affect the move or usage.
//
// It performs two functions:
//  1. Blocks moves that would violate type consistency (side effect on candidate status).
//  2. Returns true if the variable declaration must be preserved for type info,
//     even if the variable itself is unused.
func (cm CandidateManager) evaluateTypeConstraints(declarations []usage.DeclarationNode) bool {
	// Analyze the variable's declaration and usage pattern
	if len(declarations) < 2 {
		return false
	}

	firstDecl := declarations[0].Decl
	if !firstDecl.Valid() {
		return false
	}

	// Check if the declaration is a move candidate
	m, ok := cm.candidates[firstDecl]
	if !ok || !m.movable() {
		return false
	}

	if !cm.typeChange(declarations[1:]) {
		return false
	}

	if m.TargetNode != nil {
		// Apply blocking side effect
		m.Status = category.MoveBlockedTypeIncompatible
		cm.candidates[firstDecl] = m
	}

	// If the variable is unused at declaration but its type information relies on
	// the initialization, we must preserve it as "used" (not add to the unused list).
	return true
}

// typeChange finds the next non-moved usage of a variable after the first declaration.
// Returns false if no such usage exists.
func (cm CandidateManager) typeChange(declarations []usage.DeclarationNode) bool {
	for _, declaration := range declarations {
		// skip moved declarations
		if m, ok := cm.candidates[declaration.Decl]; ok && m.movable() {
			continue
		}

		return declaration.Usage.Has(usage.UsageTypeChange)
	}

	return false
}
