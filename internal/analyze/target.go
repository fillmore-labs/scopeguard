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
	"runtime/trace"
	"slices"

	"golang.org/x/tools/go/ast/inspector"
)

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
	dontFix    bool
}

// target determines which declarations can be moved to tighter scopes and where they should go.
//
// This is the third phase of the analyzer pipeline, following declaration and usage analysis.
// It takes the usage scopes identified in the previous phase and:
//  1. Applies safety constraints to avoid semantic changes (loops, function literals)
//  2. Determines the target AST node for each moveable declaration
//  3. Resolves conflicts when multiple declarations compete for the same Init field
//  4. Returns a sorted list of valid move targets
//
// Parameters:
//   - scopes: Analyzer providing scope hierarchy and target node lookup
//   - usage: Map from declaration index to its usage scope information
//
// Returns:
//   - targetResult: Sorted list of declarations that can be moved, with target nodes and flags
//   - error: Non-nil if the inspector result is missing (should never happen in practice)
func (p pass) target(ctx context.Context, in *inspector.Inspector, scopes scopeAnalyzer, usage usageResult) (targetResult, error) {
	defer trace.StartRegion(ctx, "target").End()

	initFields := make(map[ast.Node]int)

	targets := make(map[nodeIndex]targetWithFlags)

	for decl, usageScope := range usage.scopes {
		declScope := usageScope.decl
		if usageScope.usage == declScope {
			continue // Cannot move, already at the innermost scope
		}

		c := in.At(decl)
		declNode := c.Node()

		// Apply safety constraints: find the tightest scope that avoids
		// semantic hazards (loop bodies, function literals)
		safeScope := scopes.findSafeScope(declScope, usageScope.usage)
		if safeScope == nil {
			p.reportInternalError(declNode, "Invalid scope calculations")
			continue
		}

		var (
			targetNode ast.Node
			initField  bool
		)

		// Target node selection:
		switch declNode.(type) {
		case *ast.AssignStmt:
			// Including init fields (if/for/switch)
			initField, targetNode = scopes.findTargetNode(declScope, safeScope)

		case *ast.DeclStmt:
			// Only block statements
			targetNode = scopes.findTargetNodeInBlock(declScope, safeScope)

		default:
			p.reportInternalError(declNode, "Unknown declaration type %T", declNode)
			continue
		}

		if targetNode == nil {
			continue
		}

		dontFix := p.identShadowed(c, declScope, safeScope)

		if initField && !dontFix {
			initFields[targetNode]++
		}

		targets[decl] = targetWithFlags{targetNode, dontFix}
	}

	// check whether multiple declarations target the same Init field:
	//   - All conflicting moves are marked with dontFix flag
	//   - Diagnostics are still reported, but suggested fixes are omitted
	//     to avoid making an arbitrary choice.
	for decl, t := range targets {
		if count, ok := initFields[t.targetNode]; ok && count > 1 && !t.dontFix {
			t.dontFix = true
			targets[decl] = t
		}
	}

	result := targetResult{move: moveTargets(targets, usage.unused)}

	return result, nil
}

func moveTargets(targets map[nodeIndex]targetWithFlags, unused map[nodeIndex]bool) []moveTarget {
	moveTargets := make([]moveTarget, 0, len(targets)+len(unused))

	for decl, t := range targets {
		moveTargets = append(moveTargets, moveTarget{t.targetNode, decl, t.dontFix})
	}

	for decl, dontFix := range unused {
		moveTargets = append(moveTargets, moveTarget{nil, decl, dontFix})
	}

	// Sort targets in traversal order.
	slices.SortFunc(moveTargets, func(a, b moveTarget) int { return int(a.decl - b.decl) })

	return moveTargets
}
