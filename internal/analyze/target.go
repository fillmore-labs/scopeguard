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
	"iter"
	"runtime/trace"
	"slices"

	"golang.org/x/tools/go/ast/inspector"
)

// TargetOptions contains configurable options for analyzing variable scope tightening.
type TargetOptions struct {
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
}

// Targets determines which declarations can be moved to tighter scopes and where they should go.
//
// Returns a sorted list of move targets.
func Targets(ctx context.Context, p Pass, in *inspector.Inspector, opts TargetOptions) TargetResult {
	defer trace.StartRegion(ctx, "Target").End()

	targets := make(targets)

	targets.findTargets(p, in, opts)

	// Check whether a move would change the declaration type
	targets.resolveTypeChanges(opts.Usages, opts.Conservative)

	// Check whether a remaining declaration needs the type from a previous one
	targets.resolveTypeIncompatibilities(opts.Usages)

	// Check whether fixing would only leave a single target with unused declarations
	orphanedDeclarations := targets.findOrphanedDeclarations(opts.Usages)

	move := targets.moveTargets(orphanedDeclarations)

	return TargetResult{Move: move}
}

// targets maps declaration indices to their target information during the target phase.
// It provides methods for conflict resolution and conversion to the final moveTarget slice.
type targets map[NodeIndex]targetWithFlags

// targetWithFlags is an intermediate representation used during target calculation.
//
// This type is used internally by the target function to track potential moves before
// resolving conflicts. It contains the same information as moveTarget but
// without the decl index, since the map key provides that information.
//
// After conflict resolution these are converted to moveTarget instances.
type targetWithFlags struct {
	targetNode ast.Node
	unused     []string
	status     MoveStatus
}

func (t targetWithFlags) movable() bool { return t.status.Movable() }

// findTargets iterates through all usage scopes and determines valid target nodes
// for declarations that can be moved to tighter scopes. It handles:
//   - Filtering out suppressed declarations (nolint, maxLines)
//   - Finding safe scopes that avoid semantic hazards
//   - Selecting appropriate target AST nodes based on declaration type
//   - Tracking Init field conflicts
func (t targets) findTargets(p Pass, in *inspector.Inspector, opts TargetOptions) {
	initFields := make(map[ast.Node]NodeIndex)

	unused := collectUnusedVariables(opts.Usages)

	for decl, scopeRange := range opts.ScopeRanges {
		if decl == InvalidNode {
			continue
		}

		declScope, usageScope := scopeRange.Decl, scopeRange.Usage
		if usageScope == declScope {
			continue // Cannot move, already at the innermost scope
		}

		declCursor := in.At(decl)
		declNode := declCursor.Node()

		// Apply safety constraints: find the tightest scope that avoids
		// semantic hazards (loop bodies, function literals)
		safeScope := opts.FindSafeScope(declScope, usageScope)

		switch safeScope {
		case declScope:
			continue

		case nil:
			p.ReportInternalError(declNode, "Invalid scope calculations")

			continue
		}

		var (
			identifiers iter.Seq[*ast.Ident]
			onlyBlock   bool
		)

		switch n := declNode.(type) {
		case *ast.AssignStmt:
			onlyBlock = opts.MaxLines > 0 && opts.Lines(declNode) > opts.MaxLines
			identifiers = allAssigned(n)

		case *ast.DeclStmt:
			onlyBlock = true
			identifiers = allDeclared(n)

		default:
			continue
		}

		// Target node selection
		targetNode, initField := opts.FindTargetNode(declScope, safeScope, onlyBlock)

		if targetNode == nil {
			continue
		}

		if opts.HasNoLintComment(declNode.Pos()) {
			continue
		}

		target := targetWithFlags{targetNode, unused[decl], MoveAllowed}

		// Do various checks whether we should suppress the fix (but not the diagnostic).
		// They are called in order.
		switch {
		// In generated files.
		case opts.Generated():
			target.status = MoveBlockedGenerated

		// Check if a moved identifier is already declared at the target scope.
		case !initField && alreadyDeclaredInScope(safeScope, identifiers):
			target.status = MoveBlockedDeclared

		// Check if an identifier used in the definition is shadowed.
		case usedIdentifierShadowed(p.TypesInfo, declCursor, declScope, safeScope):
			target.status = MoveBlockedShadowed

		// Only one declaration can be moved to an init field.
		// This check has side effects.
		case initField && t.alreadyTargeted(initFields, targetNode, decl):
			target.status = MoveBlockedInitConflict

		// In conservative mode, we check whether there are statements between
		// the declaration and the usage to ensure no side effects are crossed.
		case opts.Conservative && checkSkippedStatements(declCursor, targetNode):
			target.status = MoveBlockedStatements
		}

		t[decl] = target
	}
}

// alreadyTargeted checks whether the init field of the target node already has a move candidate.
//
// Note that this has the side effect of memorizing the first candidate and blocking it when a we have a second one.
func (t targets) alreadyTargeted(initFields map[ast.Node]NodeIndex, targetNode ast.Node, decl NodeIndex) bool {
	firstDecl, ok := initFields[targetNode]
	if !ok {
		initFields[targetNode] = decl

		return false
	}

	// Multiple declarations target the same Init field:
	//   - All conflicting moves are marked with status moveBlocked
	//   - Diagnostics are still reported, but suggested fixes are omitted
	//     to avoid making an arbitrary choice.
	if otherTarget := t[firstDecl]; otherTarget.movable() {
		otherTarget.status = MoveBlockedInitConflict
		t[firstDecl] = otherTarget
	}

	return true
}

// resolveTypeChanges marks status as moveBlocked when the move would change
// the type of a used variable.
func (t targets) resolveTypeChanges(usages map[*types.Var][]NodeUsage, conservative bool) {
	for _, nodes := range usages {
		for _, usage := range nodes {
			if !hasUsedTypeChange(usage.Flags, conservative) {
				continue
			}

			target := t[usage.Decl]
			if !target.movable() {
				continue
			}

			target.status = MoveBlockedTypeChange

			t[usage.Decl] = target
		}
	}
}

// resolveTypeIncompatibilities prevents moves that would lose type information.
//
// When a variable is reassigned with a different type inference, moving the first
// declaration would change the variable's type. This function detects such cases
// and blocks the move to preserve the original type semantics.
func (t targets) resolveTypeIncompatibilities(usages map[*types.Var][]NodeUsage) {
	for v, nodes := range usages {
		if len(nodes) < 2 {
			continue
		}

		// Check whether the first target is a candidate
		first := nodes[0].Decl
		if first == InvalidNode {
			continue
		}

		target, ok := t[first]
		if !ok || !target.movable() {
			continue
		}

		// Skip if not being moved and the variable is not in the unused list.
		// In this case the declaration remains in place and no action is needed.
		if target.targetNode == nil && !slices.Contains(target.unused, v.Name()) {
			continue
		}

		// Skip if the next non-moved declaration does not change the type
		if next, typeChange := t.findNextUsage(nodes); next == InvalidNode || !typeChange {
			continue
		}

		// While the value is not used, the type is
		target.unused = slices.DeleteFunc(target.unused, func(name string) bool { return name == v.Name() })
		if target.targetNode != nil {
			target.status = MoveBlockedTypeIncompatible // Prevent movement
		}
		t[first] = target
	}
}

// findNextUsage finds the next non-moved usage of a variable after the first declaration.
// Returns invalidNodeIndex if no such usage exists.
func (t targets) findNextUsage(usages []NodeUsage) (NodeIndex, bool) {
	if len(usages) < 2 {
		return InvalidNode, false
	}

	for _, usage := range usages[1:] {
		decl := usage.Decl

		// skip moved declarations
		if ti, ok := t[decl]; ok && ti.movable() {
			continue
		}

		return decl, usage.Flags&UsageTypeChange != 0
	}

	return InvalidNode, false
}

// findOrphanedDeclarations identifies declarations that would become entirely unused
// after other declarations are moved. These can have all their variables replaced with '_'.
//
// This handles the case where a variable is reassigned multiple times, and moving
// the first declaration leaves subsequent assignments with no remaining reads.
func (t targets) findOrphanedDeclarations(usages map[*types.Var][]NodeUsage) map[NodeIndex][]string {
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
			if t, ok := t[index]; ok && t.movable() {
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

			if t, ok := t[index]; ok && t.movable() {
				continue
			}

			orphanedDeclarations[index] = append(orphanedDeclarations[index], v.Name())
		}
	}

	return orphanedDeclarations
}

// moveTargets converts the intermediate target map to a sorted slice of moveTarget.
func (t targets) moveTargets(orphanedDeclarations map[NodeIndex][]string) []MoveTarget {
	moveTargets := make([]MoveTarget, 0, len(t)+len(orphanedDeclarations))

	for decl, target := range t {
		moveTargets = append(moveTargets, MoveTarget{target.targetNode, target.unused, decl, target.status})
	}

	for decl, unused := range orphanedDeclarations {
		moveTargets = append(moveTargets, MoveTarget{nil, unused, decl, MoveAllowed})
	}

	// Sort targets in traversal order.
	slices.SortFunc(moveTargets, func(a, b MoveTarget) int { return int(a.Decl) - int(b.Decl) })

	return moveTargets
}
