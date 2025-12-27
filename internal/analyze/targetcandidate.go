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

package analyze

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"golang.org/x/tools/go/ast/inspector"
)

// CandidateManager manages the set of declaration move candidates.
type CandidateManager struct {
	candidates map[NodeIndex]MoveCandidate
}

func newCandidateManager() CandidateManager {
	return CandidateManager{
		candidates: make(map[NodeIndex]MoveCandidate),
	}
}

// BlockMovesWithTypeChanges marks candidates as blocked when moving would change
// the inferred type of a variable that is actually used.
//
// Type changes are blocked in two cases:
//   - Conservative mode: Any type change for a used variable
//   - Type change to untyped nil (would cause compile errors)
func (cm CandidateManager) BlockMovesWithTypeChanges(allUsages map[*types.Var][]NodeUsage, conservative bool) {
	for _, usages := range allUsages {
		for _, usage := range usages {
			if !usedAndTypeChange(usage.Flags, conservative) {
				continue
			}

			m, ok := cm.candidates[usage.Decl]
			if !ok || !m.movable() {
				continue
			}

			m.status = MoveBlockedTypeChange
			cm.candidates[usage.Decl] = m
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
func (cm CandidateManager) BlockMovesLosingTypeInfo(allUsages map[*types.Var][]NodeUsage) map[NodeIndex][]*types.Var {
	unused := make(map[NodeIndex][]*types.Var)

	for v, usages := range allUsages {
		// If type info preservation is needed, the first declaration is effectively used (for type info)
		keepTypeInfo := cm.evaluateTypeConstraints(usages)

		for _, usage := range usages {
			if keepTypeInfo {
				keepTypeInfo = false
				continue
			}

			// Populate unused map
			if usage.Flags&UsageUsed == 0 {
				unused[usage.Decl] = append(unused[usage.Decl], v)
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
func (cm CandidateManager) evaluateTypeConstraints(usages []NodeUsage) bool {
	// Analyze the variable's declaration and usage pattern
	if len(usages) < 2 {
		return false
	}

	firstDecl := usages[0].Decl
	if firstDecl == InvalidNode {
		return false
	}

	// Check if the declaration is a move candidate
	m, ok := cm.candidates[firstDecl]
	if !ok || !m.movable() {
		return false
	}

	if !cm.typeChange(usages[1:]) {
		return false
	}

	if m.targetNode != nil {
		// Apply blocking side effect
		m.status = MoveBlockedTypeIncompatible
		cm.candidates[firstDecl] = m
	}

	// If the variable is unused at declaration but its type information relies on
	// the initialization, we must preserve it as "used" (not add to the unused list).
	return true
}

// typeChange finds the next non-moved usage of a variable after the first declaration.
// Returns false if no such usage exists.
func (cm CandidateManager) typeChange(usages []NodeUsage) bool {
	for _, usage := range usages {
		// skip moved declarations
		if m, ok := cm.candidates[usage.Decl]; ok && m.movable() {
			continue
		}

		return usage.Flags&UsageTypeChange != 0
	}

	return false
}

// ResolveInitFieldConflicts handles multiple declarations targeting the same init field.
//
// If conservative mode is on, all conflicts are blocked.
// If not conservative, it attempts to combine compatible simple assignments (x:=1, y:=2 -> x,y:=1,2).
func (cm CandidateManager) ResolveInitFieldConflicts(in *inspector.Inspector, combine bool) {
	// Map to track multiple candidates for the same target node
	targets := make(map[ast.Node][]NodeIndex)

	for decl, m := range cm.candidates {
		// Only consider movable candidates
		if !m.status.Movable() {
			continue
		}

		// Check if the target is an init field
		if !initField(m.targetNode) {
			continue
		}

		targets[m.targetNode] = append(targets[m.targetNode], decl)
	}

	for _, decls := range targets {
		if len(decls) < 2 {
			continue
		}

		// Attempt to combine candidates
		if combine && combinable(in, decls) {
			// If one candidate depends on another, they aren't movable.
			cm.combine(decls)

			continue
		}

		// Block all conflicts when not combining
		for _, decl := range decls {
			m := cm.candidates[decl]
			m.status = MoveBlockedInitConflict
			cm.candidates[decl] = m
		}
	}
}

// combinable verifies all are short variable declarations with n:n assignments.
func combinable(in *inspector.Inspector, decls []NodeIndex) bool {
	for _, decl := range decls {
		c := in.At(decl)
		if stmt, ok := c.Node().(*ast.AssignStmt); !ok || stmt.Tok != token.DEFINE || len(stmt.Lhs) != len(stmt.Rhs) {
			return false
		}
	}

	return true
}

// combine combines the declarations into the first one.
func (cm CandidateManager) combine(decls []NodeIndex) {
	// Sort by declaration index to ensure a deterministic order.
	slices.Sort(decls)

	// Combine into the first candidate.
	firstDecl, additionalDecls := decls[0], decls[1:]

	// We store the additional declaration indices in the first candidate.
	m := cm.candidates[firstDecl]
	m.absorbedDecls = additionalDecls
	cm.candidates[firstDecl] = m

	// The first candidate remains MoveAllowed, additional ones are marked MoveAbsorbed.
	for _, decl := range additionalDecls {
		m := cm.candidates[decl]
		m.status = MoveAbsorbed
		cm.candidates[decl] = m
	}
}

// BlockSideEffects marks candidates as blocked if there are intervening statements with possible side effects.
func (cm CandidateManager) BlockSideEffects(in *inspector.Inspector, sc SafetyChecker) {
	for decl, m := range cm.candidates {
		// Only consider movable candidates
		if !m.movable() {
			continue
		}

		// Conservative mode - check for intervening statements with possible side effects
		if parent, start, end := enclosingInterval(in.At(decl), m.targetNode); !sc.IntervalInert(parent, m.absorbedDecls, start, end) {
			m.status = MoveBlockedStatements
			cm.candidates[decl] = m
		}
	}
}

// OrphanedDeclarations identifies declarations that would become entirely unused
// after other declarations are moved. These can have all their variables replaced with '_'.
//
// This handles the case where a variable is reassigned multiple times, and moving
// the first declaration leaves subsequent assignments with no remaining reads.
func (cm CandidateManager) OrphanedDeclarations(allUsages map[*types.Var][]NodeUsage) map[NodeIndex][]*types.Var {
	orphanedDeclarations := make(map[NodeIndex][]*types.Var)

	for v, usages := range allUsages {
		// Skip if fewer than 2 declarations (need at least one moved and one remaining)
		if len(usages) < 2 {
			continue
		}

		// Check if there are any read usages remaining
		hasUsage := false

		for _, usage := range usages {
			index := usage.Decl
			if index == InvalidNode {
				hasUsage = true
				break
			}

			// skip moved declarations
			if m, ok := cm.candidates[index]; ok && m.movable() {
				continue
			}

			if usage.Flags&UsageUsed != 0 {
				hasUsage = true
				break
			}
		}

		if hasUsage {
			continue
		}

		// No usages remaining, mark all remaining occurrences for removal
		for _, usage := range usages {
			index := usage.Decl
			if index == InvalidNode {
				continue
			}

			if m, ok := cm.candidates[index]; ok && m.movable() {
				continue
			}

			orphanedDeclarations[index] = append(orphanedDeclarations[index], v)
		}
	}

	return orphanedDeclarations
}

// SortedMoveTargets converts the intermediate candidate map to a sorted slice of MoveTarget.
//
// Combines:
//   - Regular move candidates (with or without unused variables)
//   - Orphaned declarations (no target node, all variables unused)
//
// Returns results sorted by source position for deterministic output.
func (cm CandidateManager) SortedMoveTargets(unused, orphanedDeclarations map[NodeIndex][]*types.Var) []MoveTarget {
	moveTargets := make([]MoveTarget, 0, len(cm.candidates)+len(orphanedDeclarations))

	for decl, m := range cm.candidates {
		var absorbedDecls []MovableDecl
		for _, idx := range m.absorbedDecls {
			absorbedDecls = append(absorbedDecls, MovableDecl{idx, varNames(unused[idx])})
		}

		moveTargets = append(moveTargets, MoveTarget{MovableDecl{decl, varNames(unused[decl])}, m.targetNode, absorbedDecls, m.status})
	}

	for decl, orphaned := range orphanedDeclarations {
		moveTargets = append(moveTargets, MoveTarget{MovableDecl{decl, varNames(orphaned)}, nil, nil, MoveAllowed})
	}

	// Sort targets in traversal order.
	slices.SortFunc(moveTargets, func(a, b MoveTarget) int { return int(a.Decl - b.Decl) })

	return moveTargets
}

func varNames(vars []*types.Var) []string {
	if len(vars) == 0 {
		return nil
	}

	names := make([]string, len(vars))
	for i, v := range vars {
		names[i] = v.Name()
	}

	return names
}
