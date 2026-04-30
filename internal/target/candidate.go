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
	"go/ast"
	"go/types"
	"iter"
	"slices"

	"fillmore-labs.com/scopeguard/internal/astutil"
	"fillmore-labs.com/scopeguard/internal/category"
	"fillmore-labs.com/scopeguard/internal/usage"
)

// CandidateManager manages the set of declaration move candidates.
type CandidateManager struct {
	candidates map[astutil.NodeIndex]MoveCandidate
}

// NewManager creates a [CandidateManager].
func NewManager() CandidateManager {
	return CandidateManager{
		candidates: make(map[astutil.NodeIndex]MoveCandidate),
	}
}

// AddCandidate adds candidate using the given node index and move candidate data.
func (cm CandidateManager) AddCandidate(decl astutil.NodeIndex, m MoveCandidate) {
	cm.candidates[decl] = m
}

// MoveCandidate is an intermediate representation of a potential move operation.
//
// Differences from MoveTarget:
//   - Does not include the declaration index (stored as a map key)
//   - Mutable status field (updated during conflict resolution)
type MoveCandidate struct {
	TargetNode    ast.Node            // Destination AST node (e.g., *ast.IfStmt for init field, *ast.BlockStmt for block)
	AbsorbedDecls []astutil.NodeIndex // Additional declarations merged into this one
	Status        category.MoveStatus // Whether the move is safe (MoveAllowed) or blocked (with reason)
}

func (m MoveCandidate) movable() bool { return m.Status.Movable() }

// OrphanedDeclarations identifies declarations that would become unused after
// other declarations are moved. These need their variables replaced with '_'.
//
// This handles the case where a variable is reassigned multiple times, and moving
// the first declaration leaves subsequent assignments with no remaining reads.
func (cm CandidateManager) OrphanedDeclarations(allDeclarations iter.Seq2[*types.Var, []usage.DeclarationNode]) map[astutil.NodeIndex][]*types.Var {
	orphanedDeclarations := make(map[astutil.NodeIndex][]*types.Var)

	for v, declarations := range allDeclarations {
		// Skip if fewer than 2 declarations (need at least one moved and one remaining)
		if len(declarations) < 2 {
			continue
		}

		// Check if there are any read usages remaining
		hasUsage := false

		for _, declaration := range declarations {
			index := declaration.Decl
			if !index.Valid() {
				hasUsage = true
				break
			}

			// skip moved declarations
			if m, ok := cm.candidates[index]; ok && m.movable() {
				continue
			}

			if declaration.Usage.Used() {
				hasUsage = true
				break
			}
		}

		if hasUsage {
			continue
		}

		// No usages remaining, mark all remaining occurrences for removal
		for _, declaration := range declarations {
			index := declaration.Decl
			if !index.Valid() {
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
func (cm CandidateManager) SortedMoveTargets(unused, orphanedDeclarations map[astutil.NodeIndex][]*types.Var) []MoveTarget {
	moveTargets := make([]MoveTarget, 0, len(cm.candidates)+len(orphanedDeclarations))

	for decl, m := range cm.candidates {
		var absorbedDecls []MovableDecl
		for _, idx := range m.AbsorbedDecls {
			absorbedDecls = append(absorbedDecls, MovableDecl{Decl: idx, Unused: varNames(unused[idx])})
		}

		moveTargets = append(moveTargets, MoveTarget{MovableDecl: MovableDecl{Decl: decl, Unused: varNames(unused[decl])}, TargetNode: m.TargetNode, AbsorbedDecls: absorbedDecls, MoveStatus: m.Status})
	}

	for decl, orphaned := range orphanedDeclarations {
		moveTargets = append(moveTargets, MoveTarget{MovableDecl: MovableDecl{Decl: decl, Unused: varNames(orphaned)}, TargetNode: nil, AbsorbedDecls: nil, MoveStatus: category.MoveAllowed})
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
