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

package usage

import (
	"go/ast"
	"go/token"
	"go/types"

	"fillmore-labs.com/scopeguard/internal/astutil"
)

// handleAssignedVars processes a list of expressions (LHS of an assignment) to update shadow and
// usage tracking for variables being assigned to. It filters for identified variables (ignoring blank identifiers
// and non-variable expressions) and handles deduplication to ensure a variable is processed only once per assignment
// (e.g. "x, x = 1, 2").
//
// Parameters:
//   - exprs: The list of expressions on the left-hand side of the assignment.
//   - assignmentDone: The position immediately after the assignment statement.
//   - asgn: The index of the assignment node in the AST.
func (c *collector) handleAssignedVars(exprs []ast.Expr, assignmentDone token.Pos, asgn astutil.NodeIndex) {
	uses := c.TypesInfo.Uses

	var done map[*types.Var]struct{}

	for _, expr := range exprs {
		id, ok := ast.Unparen(expr).(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}

		v, ok := uses[id].(*types.Var)
		if !ok {
			continue
		}

		if done == nil {
			done = make(map[*types.Var]struct{})
		} else if _, ok := done[v]; ok {
			// Filter out duplicate occurrences, like x, x = ...
			continue
		}

		done[v] = struct{}{}

		isDeclScope := true

		for child := range v.Parent().Children() {
			if child.Contains(id.NamePos) {
				isDeclScope = false
				break
			}
		}

		if isDeclScope {
			// Reassigned at declaration scope
			c.UpdateShadows(v, id, assignmentDone)
		} else {
			// Reassigned in a subscope
			c.UpdateShadowsWithReachability(v, id, assignmentDone)
		}

		c.TrackNestedAssignment(v, id, assignmentDone, asgn)
	}
}
