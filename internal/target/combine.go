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
	"go/token"
	"slices"

	"golang.org/x/tools/go/ast/inspector"

	"fillmore-labs.com/scopeguard/internal/astutil"
	"fillmore-labs.com/scopeguard/internal/category"
)

// ResolveInitFieldConflicts handles multiple declarations targeting the same init field.
//
// It attempts to combine compatible simple assignments (x := 1; y := 2 -> x, y := 1, 2).
func (cm CandidateManager) ResolveInitFieldConflicts(in *inspector.Inspector, combine bool) {
	// Map to track multiple candidates for the same target node
	targets := make(map[ast.Node][]astutil.NodeIndex)

	for decl, m := range cm.candidates {
		// Only consider movable candidates.
		// If one statement depends on a previous statement, the previous statement is unmovable.
		if !m.Status.Movable() {
			continue
		}

		// Check if the target is an init field
		if !initField(m.TargetNode) {
			continue
		}

		targets[m.TargetNode] = append(targets[m.TargetNode], decl)
	}

	for _, decls := range targets {
		if len(decls) < 2 {
			continue
		}

		if combine && combinable(in, decls) {
			cm.combine(decls) // Combine candidates
		} else {
			cm.setStatus(decls, category.MoveBlockedInitConflict) // Block all conflicts when not combining
		}
	}
}

func (cm CandidateManager) setStatus(decls []astutil.NodeIndex, status category.MoveStatus) {
	for _, decl := range decls {
		m := cm.candidates[decl]
		m.Status = status
		cm.candidates[decl] = m
	}
}

// initField determines whether the targetNode is an initialization field in a control structure.
func initField(targetNode ast.Node) bool {
	switch targetNode.(type) {
	case *ast.IfStmt,
		*ast.ForStmt,
		*ast.SwitchStmt,
		*ast.TypeSwitchStmt:
		return true

	default:
		return false
	}
}

// combinable verifies all are short variable declarations with n:n assignments.
func combinable(in *inspector.Inspector, decls []astutil.NodeIndex) bool {
	for _, decl := range decls {
		if stmt, ok := decl.Node(in).(*ast.AssignStmt); !ok ||
			stmt.Tok != token.DEFINE || len(stmt.Lhs) != len(stmt.Rhs) {
			return false // Not a short declaration with separate variables
		}
	}

	return true
}

// combine combines the declarations into the first one.
func (cm CandidateManager) combine(decls []astutil.NodeIndex) {
	// Sort by declaration index to ensure a deterministic order.
	slices.Sort(decls)

	// Combine into the first candidate.
	firstDecl, additionalDecls := decls[0], decls[1:]

	// We store the additional declaration indices in the first candidate.
	m := cm.candidates[firstDecl]
	m.AbsorbedDecls = additionalDecls
	cm.candidates[firstDecl] = m

	// The first candidate remains, additional ones are absorbed.
	cm.deleteCandidates(additionalDecls)
}

func (cm CandidateManager) deleteCandidates(decls []astutil.NodeIndex) {
	for _, decl := range decls {
		delete(cm.candidates, decl)
	}
}
