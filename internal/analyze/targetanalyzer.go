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
	"context"
	"go/ast"
	"go/token"
	"runtime/trace"

	"golang.org/x/tools/go/ast/inspector"
)

// TargetAnalyzer contains configurable options for analyzing variable scope tightening.
type TargetAnalyzer struct {
	// Pass augments the current [*analysis.Pass]
	Pass

	// TargetScope provides context for scope adjustments and safety checks.
	TargetScope

	// SafetyChecker validates potential variable moves.
	SafetyChecker

	// MaxLines specifies the maximum number of lines a declaration can span to be considered for moving
	// into control flow initializers.
	MaxLines int

	// Conservative specifies to only permit moves that don't cross code with potential side effects.
	Conservative bool

	// Combine determines whether to attempt combining initialization statements during scope tightening.
	Combine bool
}

// Analyze determines which declarations can be moved to tighter scopes and where they should go.
//
// Returns a sorted list of move targets.
func (ta TargetAnalyzer) Analyze(ctx context.Context, cf CurrentFile, body inspector.Cursor, usageResult UsageData) []MoveTarget {
	defer trace.StartRegion(ctx, "Target").End()

	in := body.Inspector()

	// Identify all potential move candidates
	cm := ta.CollectMoveCandidates(body, cf, usageResult.ScopeRanges)

	// Block moves that would change variable types
	cm.BlockMovesWithTypeChanges(usageResult.Usages, ta.Conservative)

	// Calculate unused identifiers and block moves that would lose necessary type information
	unused := cm.BlockMovesLosingTypeInfo(usageResult.Usages)

	// Resolve Init field conflicts (possibly by combining them)
	cm.ResolveInitFieldConflicts(in, ta.Combine)

	if ta.Conservative {
		// In conservative mode, blocks moves if there are intervening statements with possible side effects.
		cm.BlockSideEffects(in, ta.SafetyChecker)
	}

	// Find declarations that become orphaned after other moves
	orphanedDeclarations := cm.OrphanedDeclarations(usageResult.Usages)

	// Convert candidates to the final sorted result
	return cm.SortedMoveTargets(unused, orphanedDeclarations)
}

// CollectMoveCandidates iterates through all usage scopes and determines valid target nodes
// for declarations that can be moved to tighter scopes.
func (ta TargetAnalyzer) CollectMoveCandidates(body inspector.Cursor, cf CurrentFile, scopeRanges map[NodeIndex]ScopeRange) CandidateManager {
	labels := sortedLabels(body)

	cm := newCandidateManager()

	in := body.Inspector()
	for decl, scopeRange := range scopeRanges {
		if m, ok := ta.analyzeCandidate(in, cf, decl, scopeRange, labels); ok {
			cm.candidates[decl] = m
		}
	}

	return cm
}

// analyzeCandidate evaluates a single declaration to see if it can be moved.
// It handles:
//   - Filtering out suppressed declarations (nolint, maxLines)
//   - Finding safe scopes that avoid semantic hazards
//   - Selecting appropriate target AST nodes based on the declaration type
func (ta TargetAnalyzer) analyzeCandidate(in *inspector.Inspector, cf CurrentFile, decl NodeIndex, scopeRange ScopeRange, labels []token.Pos) (MoveCandidate, bool) {
	if decl == InvalidNode {
		return MoveCandidate{}, false
	}

	declScope, usageScope := scopeRange.Decl, scopeRange.Usage
	if usageScope == declScope {
		return MoveCandidate{}, false // Cannot move, already at the innermost scope
	}

	declCursor := in.At(decl)
	declNode := declCursor.Node()

	// Find the tightest scope we can move to (avoiding loops, closures)
	safeScope := ta.FindSafeScope(declScope, usageScope)
	switch safeScope {
	case nil:
		ta.ReportInternalError(declNode, "Invalid scope calculations")
		return MoveCandidate{}, false

	case declScope: // No scope tightening possible
		return MoveCandidate{}, false
	}

	// Determine assigned identifiers and whether the declaration can be moved to an init field
	identifiers, onlyBlock := declInfo(declNode, cf, ta.MaxLines)
	if identifiers == nil {
		return MoveCandidate{}, false // Unsupported declaration type
	}

	declPos := declNode.Pos()

	// Find the nearest label after this declaration.
	// We cannot move the declaration past it to avoid placing it inside a loop.
	labelBarrier := nextLabel(labels, declPos)

	// Find the target AST node for the move
	targetNode := ta.TargetNode(declScope, safeScope, labelBarrier, onlyBlock)
	if targetNode == nil || cf.NoLintComment(declPos) {
		return MoveCandidate{}, false
	}

	// Create a move candidate
	m := MoveCandidate{targetNode: targetNode, status: MoveAllowed}

	// Do various safety checks whether we should suppress the fix (but not the diagnostic).
	if cf.Generated() {
		m.status = MoveBlockedGenerated
	} else {
		m.status = ta.Check(declCursor, declScope, safeScope, identifiers)
	}

	return m, true
}

// MoveCandidate is an intermediate representation of a potential move operation.
//
// Differences from MoveTarget:
//   - Does not include the declaration index (stored as a map key)
//   - Mutable status field (updated during conflict resolution)
type MoveCandidate struct {
	targetNode    ast.Node    // Destination AST node (e.g., *ast.IfStmt for init field, *ast.BlockStmt for block)
	status        MoveStatus  // Whether the move is safe (MoveAllowed) or blocked (with reason)
	absorbedDecls []NodeIndex // Additional declarations merged into this one
}

func (m MoveCandidate) movable() bool { return m.status.Movable() }
