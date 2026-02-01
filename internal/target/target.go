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
	"context"
	"go/ast"
	"go/token"
	"go/types"
	"iter"
	"runtime/trace"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"

	"fillmore-labs.com/scopeguard/internal/astutil"
	"fillmore-labs.com/scopeguard/internal/config"
	"fillmore-labs.com/scopeguard/internal/scope"
	"fillmore-labs.com/scopeguard/internal/target/check"
	"fillmore-labs.com/scopeguard/internal/usage"
)

// Stage contains configurable options for analyzing variable scope tightening.
type Stage struct {
	// The current [*analysis.Pass]
	*analysis.Pass

	// TargetScope provides context for scope adjustments and safety checks.
	scope.TargetScope

	// maxLines specifies the maximum number of lines a declaration can span to be considered for moving
	// into control flow initializers.
	maxLines int

	// behavior holds layout and behavioral options.
	behavior config.Behavior
}

// New creates a [target.Stage].
func New(p *analysis.Pass, scopes scope.Index, maxlines int, behavior config.Behavior) Stage {
	return Stage{
		Pass:        p,
		TargetScope: scope.NewTargetScope(scopes),
		maxLines:    maxlines,
		behavior:    behavior,
	}
}

// SelectTargets determines which declarations can be moved to tighter scopes and where they should go.
//
// Returns a sorted list of move targets.
func (ts Stage) SelectTargets(ctx context.Context, cf astutil.CurrentFile, body inspector.Cursor, usageData usage.Result) []MoveTarget {
	defer trace.StartRegion(ctx, "Target").End()

	conservate := ts.behavior.Enabled(config.Conservative)
	combine := ts.behavior.Enabled(config.CombineDeclarations)

	in := body.Inspector()

	// Identify all potential move candidates
	cm := ts.CollectMoveCandidates(body, cf, usageData.AllScopeRanges())

	// Block moves that would change variable types
	cm.BlockMovesWithTypeChanges(usageData.AllDeclarations(), conservate)

	// Calculate unused identifiers and block moves that would lose necessary type information
	unused := cm.BlockMovesLosingTypeInfo(usageData.AllDeclarations())

	// Resolve Init field conflicts (possibly by combining them)
	cm.ResolveInitFieldConflicts(in, combine)

	if conservate {
		// In conservative mode, blocks moves if there are intervening statements with possible side effects.
		cm.BlockSideEffects(ts.TypesInfo, body)
	}

	// Find declarations that become orphaned after other moves
	orphanedDeclarations := cm.OrphanedDeclarations(usageData.AllDeclarations())

	// Convert candidates to the final sorted result
	return cm.SortedMoveTargets(unused, orphanedDeclarations)
}

// CollectMoveCandidates iterates through all usage scopes and determines valid target nodes
// for declarations that can be moved to tighter scopes.
func (ts Stage) CollectMoveCandidates(body inspector.Cursor, cf astutil.CurrentFile, scopeRanges iter.Seq2[astutil.NodeIndex, usage.ScopeRange]) CandidateManager {
	labels := sortedLabels(body)

	cm := newCandidateManager()

	in := body.Inspector()

	for decl, scopeRange := range scopeRanges {
		if !decl.Valid() {
			continue
		}

		declScope, usageScope := scopeRange.Decl, scopeRange.Usage
		if usageScope == declScope {
			continue // Cannot move, already at the innermost scope
		}

		if m, ok := ts.analyzeCandidate(in, cf, decl, declScope, usageScope, labels); ok {
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
func (ts Stage) analyzeCandidate(in *inspector.Inspector, cf astutil.CurrentFile, decl astutil.NodeIndex, declScope, usageScope *types.Scope, labels []token.Pos) (MoveCandidate, bool) {
	declCursor := decl.Cursor(in)
	declNode := declCursor.Node()

	// Find the tightest scope we can move to (avoiding loops, closures)
	safeScope := ts.FindSafeScope(declScope, usageScope)
	switch safeScope {
	case nil:
		astutil.InternalError(ts.Pass, declNode, "Invalid scope calculations")
		return MoveCandidate{}, false

	case declScope: // No scope tightening possible
		return MoveCandidate{}, false
	}

	// Determine assigned identifiers and whether the declaration can be moved to an init field
	identifiers, onlyBlock := declInfo(declNode, cf, ts.maxLines)
	if identifiers == nil {
		return MoveCandidate{}, false // Unsupported declaration type
	}

	declPos := declNode.Pos()

	// Find the nearest label after this declaration.
	// We cannot move the declaration past it to avoid placing it inside a loop.
	labelBarrier := nextLabel(labels, declPos)

	// Find the target AST node for the move
	targetNode := ts.TargetNode(declScope, safeScope, labelBarrier, onlyBlock)
	if targetNode == nil || cf.NoLintComment(declPos) {
		return MoveCandidate{}, false
	}

	// Create a move candidate
	m := MoveCandidate{targetNode: targetNode, status: check.MoveAllowed}

	// Do various safety checks whether we should suppress the fix (but not the diagnostic).
	if cf.Generated() {
		m.status = check.MoveBlockedGenerated
	} else {
		m.status = check.SafetyCheck(ts.TypesInfo, declCursor, declScope, safeScope, identifiers)
	}

	return m, true
}

// declInfo extracts assigned identifiers and whether the move is restricted to block statements only.
func declInfo(declNode ast.Node, cf astutil.CurrentFile, maxLines int) (identifiers iter.Seq[*ast.Ident], onlyBlock bool) {
	switch n := declNode.(type) {
	case *ast.AssignStmt:
		// Short declarations can go to init fields if they're small enough
		return astutil.AllAssigned(n), maxLines > 0 && cf.Lines(declNode) > maxLines

	case *ast.DeclStmt:
		// var declarations can only go to block statements (not init fields)
		return astutil.AllDeclared(n), true

	default:
		// Unsupported declaration type
		return nil, false
	}
}
