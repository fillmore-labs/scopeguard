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

// handleAssignedVars processes a list of expressions (LHS of an assignment) to extract
// variables being assigned to. It filters for identified variables (ignoring blank identifiers
// and non-variable expressions) and delegates tracking to trackVars.
//
// Parameters:
//   - exprs: The list of expressions on the left-hand side of the assignment.
//   - assignmentDone: The position immediately after the assignment statement.
//   - asgn: The index of the assignment node in the AST.
func (c *collector) handleAssignedVars(exprs []ast.Expr, assignmentDone token.Pos, asgn astutil.NodeIndex) {
	uses := c.TypesInfo.Uses

	vars := make([]assignedVar, 0, len(exprs))

	for _, expr := range exprs {
		id, ok := ast.Unparen(expr).(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}

		v, ok := uses[id].(*types.Var)
		if !ok {
			continue
		}

		vars = append(vars, assignedVar{v, id})
	}

	c.trackVars(vars, assignmentDone, asgn)
}

// trackVars updates shadow and usage tracking for a list of assigned variables.
// It handles deduplication to ensure a variable is processed only once per assignment
// (e.g. "x, x = 1, 2").
//
// Parameters:
//   - vars: The list of variables extracted from the assignment.
//   - assignmentDone: The position where the assignment is considered complete.
//   - asgn: The index of the assignment node.
func (c *collector) trackVars(vars []assignedVar, assignmentDone token.Pos, asgn astutil.NodeIndex) {
	if len(vars) == 0 {
		return
	}

	done := make(map[*types.Var]struct{})
	for _, vid := range vars {
		// Filter out duplicate occurrences, like x, x = ...
		if _, ok := done[vid.Var]; ok {
			continue
		}

		done[vid.Var] = struct{}{}

		c.UpdateShadows(vid.Var, vid.Ident, assignmentDone)

		c.TrackAssignment(vid.Var, vid.Ident, assignmentDone, asgn)
	}
}

// assignedVar captures a variable and the specific identifier used in an assignment.
type assignedVar struct {
	// Var is the type-checked variable object.
	*types.Var
	// Ident is the AST identifier node corresponding to this variable usage.
	*ast.Ident
}
