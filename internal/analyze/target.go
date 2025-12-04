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

// target determines which declarations can be moved to tighter scopes and where they should go.
//
// Returns a sorted list of move targets.
func (p pass) target(ctx context.Context, in *inspector.Inspector, file *ast.File, generated bool, scopes scopeAnalyzer, usageScopes usageResult, maxLines int) targetResult {
	defer trace.StartRegion(ctx, "target").End()

	targets := p.findTargets(in, file, generated, scopes, usageScopes, maxLines)

	// Check whether a remaining declaration needs the type from a previous one
	targets.resolveTypeIncompatibilities(in, p.TypesInfo, usageScopes.usages)

	// Check whether fixing would only leave a single target with unused declarations
	orphanedDeclarations := targets.findOrphanedDeclarations(usageScopes.usages)

	return targetResult{move: targets.moveTargets(orphanedDeclarations)}
}

// findTargets iterates through all usage scopes and determines valid target nodes
// for declarations that can be moved to tighter scopes. It handles:
//   - Filtering out suppressed declarations (nolint, maxLines)
//   - Finding safe scopes that avoid semantic hazards
//   - Selecting appropriate target AST nodes based on declaration type
//   - Tracking Init field conflicts for later resolution
func (p pass) findTargets(in *inspector.Inspector, file *ast.File, generated bool, scopes scopeAnalyzer, usageScopes usageResult, maxLines int) targets {
	targets := make(targets)
	initFields := make(map[ast.Node]int)

	lf := lineFinder{p.Fset.File(file.FileStart)}
	unused := collectUnusedVariables(usageScopes.usages)

	for decl, usageScope := range usageScopes.scopeRanges {
		declScope := usageScope.decl

		if usageScope.usage == declScope {
			continue // Cannot move, already at the innermost scope
		}

		c := in.At(decl)
		declNode := c.Node()

		if lf.hasNoLintComment(file, declNode) {
			continue
		}

		if maxLines >= 0 && lf.lines(declNode) > maxLines {
			continue
		}

		// Apply safety constraints: find the tightest scope that avoids
		// semantic hazards (loop bodies, function literals)
		safeScope := scopes.findSafeScope(declScope, usageScope.usage)

		switch safeScope {
		case declScope:
			continue

		case nil:
			p.reportInternalError(declNode, "Invalid scope calculations")

			continue
		}

		var (
			targetNode  ast.Node
			initField   bool
			identifiers iter.Seq[*ast.Ident]
		)

		// Target node selection:
		switch n := declNode.(type) {
		case *ast.AssignStmt:
			// Including init fields (if/for/switch)
			targetNode, initField = scopes.findTargetNode(declScope, safeScope)
			identifiers = allAssigned(n)

		case *ast.DeclStmt:
			// Only block statements
			targetNode = scopes.findTargetNodeInBlock(declScope, safeScope)
			identifiers = allDeclared(n)

		default:
			p.reportInternalError(declNode, "Unknown declaration type %T", declNode)
			continue
		}

		if targetNode == nil {
			continue
		}

		// Do various checks whether we should suppres the fix (but not the diagnostic):
		// - In generated files
		// - When the identifier is alread declared at the target scope
		// - When an identifier used in the definition is shadowed by a different one
		// Note that we don't care about changes of variables we depend on. This will potentially break logic, but not compilation.
		dontFix := generated

		if !initField && !dontFix {
			dontFix = alreadyDeclaredInScope(safeScope, identifiers)
		}

		if !dontFix {
			dontFix = usedIdentifierShadowed(p.TypesInfo, c, declNode, declScope, safeScope)
		}

		targets[decl] = targetWithFlags{targetNode, unused[decl], dontFix}

		if initField && !dontFix {
			initFields[targetNode]++
		}
	}

	// Check whether multiple declarations target the same Init field:
	//   - All conflicting moves are marked with dontFix flag
	//   - Diagnostics are still reported, but suggested fixes are omitted
	//     to avoid making an arbitrary choice.
	targets.resolveInitFieldConflicts(initFields)

	return targets
}

// collectUnusedVariables builds a map from declaration indices to the names of
// variables that are declared but never read at that declaration site.
func collectUnusedVariables(usages map[*types.Var][]nodeUsage) map[nodeIndex][]string {
	unused := make(map[nodeIndex][]string)

	for v, nodes := range usages {
		for _, usage := range nodes {
			if usage.used {
				continue
			}

			unused[usage.decl] = append(unused[usage.decl], v.Name())
		}
	}

	return unused
}

// targets maps declaration indices to their target information during the target phase.
// It provides methods for conflict resolution and conversion to the final moveTarget slice.
type targets map[nodeIndex]targetWithFlags

// targetWithFlags is an intermediate representation used during target calculation.
//
// This type is used internally by the target function to track potential moves before
// resolving Init field conflicts. It contains the same information as moveTarget but
// without the decl index, since the map key provides that information.
//
// After conflict resolution (where multiple variables competing for the same Init field
// are marked with dontFix), these are converted to moveTarget instances.
type targetWithFlags = struct {
	targetNode ast.Node
	unused     []string
	dontFix    bool
}

// resolveInitFieldConflicts marks declarations as dontFix when multiple
// declarations target the same Init field (if/for/switch). This avoids
// making an arbitrary choice about which declaration to move.
func (t targets) resolveInitFieldConflicts(initFields map[ast.Node]int) {
	for decl, target := range t {
		if count, ok := initFields[target.targetNode]; ok && count > 1 && !target.dontFix {
			target.dontFix = true
			t[decl] = target
		}
	}
}

// moveTargets converts the intermediate target map to a sorted slice of moveTarget.
func (t targets) moveTargets(orphanedDeclarations map[nodeIndex][]string) []moveTarget {
	moveTargets := make([]moveTarget, 0, len(t)+len(orphanedDeclarations))

	for decl, target := range t {
		moveTargets = append(moveTargets, moveTarget{target.targetNode, target.unused, decl, target.dontFix})
	}

	for decl, unused := range orphanedDeclarations {
		moveTargets = append(moveTargets, moveTarget{nil, unused, decl, false})
	}

	// Sort targets in traversal order.
	slices.SortFunc(moveTargets, func(a, b moveTarget) int { return int(a.decl) - int(b.decl) })

	return moveTargets
}
