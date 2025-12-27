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
	"go/token"
	"go/types"
	"runtime/trace"
	"slices"

	"golang.org/x/tools/go/ast/inspector"
)

// TargetAnalyzer contains configurable options for analyzing variable scope tightening.
type TargetAnalyzer struct {
	// Pass augments the current [*analysis.Pass]
	Pass

	// Inspector provides access to the AST.
	Body inspector.Cursor

	// ScopeAnalyzer provides context for scope adjustments and safety checks.
	ScopeAnalyzer

	// CurrentFile contains file-level information for scope and lint assessments.
	CurrentFile

	// UsageResult holds usage details for variables within analyzed scopes.
	UsageResult

	// MaxLines specifies the maximum number of lines a declaration can span to be considered for moving
	// into control flow initializers.
	MaxLines int

	// Conservative specifies to only permit moves that don't cross code with potential side effects.
	Conservative bool

	// Combine determines whether to attempt combining initialization statements during scope tightening.
	Combine bool
}

// Targets determines which declarations can be moved to tighter scopes and where they should go.
//
// Returns a sorted list of move targets.
func (tm TargetAnalyzer) Targets(ctx context.Context) TargetResult {
	defer trace.StartRegion(ctx, "Targets").End()

	// Identify all potential move candidates
	cm := tm.CollectMoveCandidates()

	// Block moves that would change variable types
	cm.BlockMovesWithTypeChanges(tm.Usages, tm.Conservative)

	// Block moves that would lose necessary type information
	cm.BlockMovesLosingTypeInfo(tm.Usages)

	// Resolve Init field conflicts (possibly by combining them)
	combine := tm.Combine && !tm.Conservative
	cm.ResolveInitFieldConflicts(tm.Body.Inspector(), combine)

	if tm.Conservative {
		// In conservative mode, blocks moves if there are intervening statements with possible side effects.
		cm.BlockSideEffects(tm.Body.Inspector(), tm.TypesInfo)
	}

	// Find declarations that become orphaned after other moves
	orphanedDeclarations := cm.OrphanedDeclarations(tm.Usages)

	// Convert candidates to the final sorted result
	move := cm.SortedMoveTargets(orphanedDeclarations)

	return TargetResult{Move: move}
}

// CollectMoveCandidates iterates through all usage scopes and determines valid target nodes
// for declarations that can be moved to tighter scopes.
func (tm TargetAnalyzer) CollectMoveCandidates() CandidateManager {
	cm := newCandidateManager()

	unused := collectUnusedVariables(tm.Usages)

	labels := sortedLabels(tm.Body)

	for decl, scopeRange := range tm.ScopeRanges {
		if m, ok := tm.analyzeCandidate(decl, scopeRange, labels); ok {
			m.unused = unused[decl]
			cm.candidates[decl] = m
		}
	}

	return cm
}

// CandidateManager manages the set of declaration move candidates.
type CandidateManager struct {
	candidates map[NodeIndex]MoveCandidate
}

func newCandidateManager() CandidateManager {
	return CandidateManager{
		candidates: make(map[NodeIndex]MoveCandidate),
	}
}

// MoveCandidate is an intermediate representation of a potential move operation.
//
// Differences from MoveTarget:
//   - Does not include the declaration index (stored as a map key)
//   - Mutable status field (updated during conflict resolution)
type MoveCandidate struct {
	targetNode      ast.Node    // Destination AST node (e.g., *ast.IfStmt for init field, *ast.BlockStmt for block)
	unused          []string    // Variable names that are unused in this declaration
	status          MoveStatus  // Whether the move is safe (MoveAllowed) or blocked (with reason)
	additionalDecls []NodeIndex // Additional declarations merged into this one
}

func (m MoveCandidate) movable() bool { return m.status.Movable() }

// analyzeCandidate evaluates a single declaration to see if it can be moved.
// It handles:
//   - Filtering out suppressed declarations (nolint, maxLines)
//   - Finding safe scopes that avoid semantic hazards
//   - Selecting appropriate target AST nodes based on the declaration type
func (tm TargetAnalyzer) analyzeCandidate(
	decl NodeIndex,
	scopeRange ScopeRange,
	labels []token.Pos,
) (MoveCandidate, bool) {
	if decl == InvalidNode {
		return MoveCandidate{}, false
	}

	declScope, usageScope := scopeRange.Decl, scopeRange.Usage
	if usageScope == declScope {
		return MoveCandidate{}, false // Cannot move, already at the innermost scope
	}

	declCursor := tm.Body.Inspector().At(decl)
	declNode := declCursor.Node()
	declPos := declNode.Pos()

	// Find the tightest scope we can move to (avoiding loops, closures)
	safeScope := tm.FindSafeScope(declScope, usageScope)
	if !isScopeMoveValid(tm.Pass, safeScope, declScope, declNode) {
		return MoveCandidate{}, false
	}

	// Find the nearest label at or after this declaration.
	// If a label exists, it acts as a barrier - we cannot move the declaration
	// past it to avoid placing it inside a goto loop.
	labelBarrier := nextLabel(labels, declPos)

	// Determine assigned identifiers and whether the declaration can be moved to an init field
	identifiers, onlyBlock := declInfo(declNode, tm.CurrentFile, tm.MaxLines)
	if identifiers == nil {
		return MoveCandidate{}, false // Unsupported declaration type
	}

	// Find the target AST node for the move
	targetNode := tm.TargetNode(declScope, safeScope, labelBarrier, onlyBlock)
	if targetNode == nil || tm.NoLintComment(declPos) {
		return MoveCandidate{}, false
	}

	// Create a move candidate
	m := MoveCandidate{targetNode: targetNode, status: MoveAllowed}

	// Do various safety checks whether we should suppress the fix (but not the diagnostic).
	// They are called in order.
	switch {
	case tm.Generated():
		// Generated files (always block fixes in generated code)
		m.status = MoveBlockedGenerated

	case alreadyDeclaredInScope(safeScope, identifiers):
		// An identifier is already declared at the target scope (would cause compile error)
		m.status = MoveBlockedDeclared

	case usedIdentifierShadowed(tm.TypesInfo, declCursor, declScope, safeScope):
		// A used identifier would be shadowed (can cause compile error or would change semantics)
		m.status = MoveBlockedShadowed
	}

	return m, true
}

func nextLabel(labels []token.Pos, pos token.Pos) token.Pos {
	l := len(labels)
	if l == 0 {
		return token.NoPos
	}

	if i, _ := slices.BinarySearch(labels, pos); i < l {
		return labels[i]
	}

	return token.NoPos
}

// sortedLabels collects all labeled statement positions in the function body.
//
// These positions are used to prevent declaration moves across labels,
// which could change program semantics.
//
// Returns nil if no labels are found, otherwise returns sorted positions.
func sortedLabels(body inspector.Cursor) []token.Pos {
	var labels []token.Pos
	for l := range body.Preorder((*ast.LabeledStmt)(nil)) {
		labels = append(labels, l.Node().Pos())
	}

	if len(labels) == 0 {
		return nil
	}

	// Sort positions to enable binary search during candidate analysis.
	// While Preorder traverses in depth-first order (which typically matches source order),
	// explicit sorting ensures correctness and is negligible overhead.
	slices.Sort(labels)

	return labels
}

// BlockMovesWithTypeChanges marks candidates as blocked when moving would change
// the inferred type of a variable that is actually used.
//
// Type changes are blocked in two cases:
//   - Conservative mode: Any type change for a used variable
//   - Type change to untyped nil (would cause compile errors)
func (cm CandidateManager) BlockMovesWithTypeChanges(usages map[*types.Var][]NodeUsage, conservative bool) {
	for _, nodes := range usages {
		for _, usage := range nodes {
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
// Moving the first declaration would change x's type from any to int.
// This function detects and blocks such moves.
func (cm CandidateManager) BlockMovesLosingTypeInfo(usages map[*types.Var][]NodeUsage) {
	for v, nodes := range usages {
		if len(nodes) < 2 {
			continue
		}

		// Check whether the first usage is a candidate for moving.
		firstDecl := nodes[0].Decl
		if firstDecl == InvalidNode {
			continue
		}

		candidate, ok := cm.candidates[firstDecl]
		if !ok || !candidate.movable() {
			continue
		}

		// Skip if not being moved and the variable is not in the unused list.
		// In this case the declaration remains in place and no action is needed.
		varName := v.Name()
		if candidate.targetNode == nil && !slices.Contains(candidate.unused, varName) {
			continue
		}

		// Skip if the next non-moved declaration does not change the type
		if !cm.typeChange(nodes) {
			continue
		}

		// While the value is not used, the type information IS needed.
		// Remove from the unused list to preserve the declaration.
		candidate.unused = slices.DeleteFunc(candidate.unused, func(name string) bool { return name == varName })
		if candidate.targetNode != nil {
			candidate.status = MoveBlockedTypeIncompatible // Prevent movement
		}

		cm.candidates[firstDecl] = candidate
	}
}

// typeChange finds the next non-moved usage of a variable after the first declaration.
// Returns false if no such usage exists.
func (cm CandidateManager) typeChange(usages []NodeUsage) bool {
	if len(usages) < 2 {
		return false
	}

	for _, usage := range usages[1:] {
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

		// Check if target is an init field
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
	// Sort by declaration index to ensure deterministic order.
	slices.Sort(decls)

	// Combine into the first candidate.
	firstDecl, additionalDecls := decls[0], decls[1:]

	// We store the additional declaration indices in the first candidate.
	m := cm.candidates[firstDecl]
	m.additionalDecls = additionalDecls
	cm.candidates[firstDecl] = m

	// The first candidate remains MoveAllowed, additional ones are marked MoveAbsorbed.
	for _, decl := range additionalDecls {
		m := cm.candidates[decl]
		m.status = MoveAbsorbed
		cm.candidates[decl] = m
	}
}

// BlockSideEffects marks candidates as blocked if there are intervening statements with possible side effects.
func (cm CandidateManager) BlockSideEffects(in *inspector.Inspector, info *types.Info) {
	for decl, m := range cm.candidates {
		// Only consider movable candidates
		if !m.movable() {
			continue
		}

		// Conservative mode - check for intervening statements with possible side effects
		if parent, start, end := enclosingInterval(in.At(decl), m.targetNode); SideEffectsInInterval(info, parent, start, end) {
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
func (cm CandidateManager) OrphanedDeclarations(usages map[*types.Var][]NodeUsage) map[NodeIndex][]string {
	orphanedDeclarations := make(map[NodeIndex][]string)

	for v, nodes := range usages {
		// Skip if fewer than 2 declarations (need at least one moved and one remaining)
		if len(nodes) < 2 {
			continue
		}

		// Check if there are any read usages remaining
		hasUsage := false

		for _, usage := range nodes {
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
		for _, usage := range nodes {
			index := usage.Decl
			if index == InvalidNode {
				continue
			}

			if m, ok := cm.candidates[index]; ok && m.movable() {
				continue
			}

			orphanedDeclarations[index] = append(orphanedDeclarations[index], v.Name())
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
func (cm CandidateManager) SortedMoveTargets(orphanedDeclarations map[NodeIndex][]string) []MoveTarget {
	moveTargets := make([]MoveTarget, 0, len(cm.candidates)+len(orphanedDeclarations))

	for decl, m := range cm.candidates {
		moveTargets = append(moveTargets, MoveTarget{m.targetNode, m.unused, decl, m.additionalDecls, m.status})
	}

	for decl, unused := range orphanedDeclarations {
		moveTargets = append(moveTargets, MoveTarget{nil, unused, decl, nil, MoveAllowed})
	}

	// Sort targets in traversal order.
	slices.SortFunc(moveTargets, func(a, b MoveTarget) int { return int(a.Decl) - int(b.Decl) })

	return moveTargets
}
