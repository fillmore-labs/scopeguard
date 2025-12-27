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

package check

import (
	"go/ast"
	"go/token"
	"go/types"

	"fillmore-labs.com/scopeguard/internal/astutil"
)

// NestedChecker tracks nested variable assignments.
type NestedChecker struct {
	// assigned maps variable assignment nesting.
	assigned map[*types.Var]assignScope

	// nestedAssigned collects usage of variables assigned during an assignment.
	nestedAssigned []NestedAssign
}

// assignScope contains information about variable nested assignment.
type assignScope struct {
	asgn astutil.NodeIndex
	end  token.Pos
}

// NewNestedChecker creates a new NestedChecker instance.
//
// If enabled is false, nested assignment tracking is disabled and the checker is a no-op that uses minimal memory.
func NewNestedChecker(enabled bool) NestedChecker {
	var nc NestedChecker

	if enabled {
		nc.assigned = make(map[*types.Var]assignScope)
	}

	return nc
}

// NestedAssigned returns the list of variables that were assigned during an assignment.
func (nc *NestedChecker) NestedAssigned() []NestedAssign {
	return nc.nestedAssigned
}

// TrackAssignment identifies nested assignments of variables and tracks their occurrences.
func (nc *NestedChecker) TrackAssignment(v *types.Var, id *ast.Ident, assignmentDone token.Pos, asgn astutil.NodeIndex) {
	if nc.assigned == nil {
		return
	}

	if assignment, ok := nc.assigned[v]; ok && id.NamePos < assignment.end {
		nc.nestedAssigned = append(nc.nestedAssigned, NestedAssign{Ident: id, Asgn: assignment.asgn})

		return
	}

	nc.assigned[v] = assignScope{asgn: asgn, end: assignmentDone}
}
