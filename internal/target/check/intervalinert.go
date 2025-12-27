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
	"slices"

	"golang.org/x/tools/go/ast/inspector"

	"fillmore-labs.com/scopeguard/internal/astutil"
)

// IntervalInert checks whether the execution interval [start, end) is inert.
//
// An interval is considered inert if it contains no statements that might have
// side effects or observable interactions with the moved code.
//
// Specifically, it returns false if the interval contains:
//   - Assignments or reassignments to existing variables (side effects)
//   - Function calls or other expressions that are not pure/constant
//   - Branching or control flow statements (other than those implicitly handled)
//
// Pure declarations (var, const, type) and short variable declarations of *new*
// variables initialized with constant expressions and no function calls are
// considered inert.
//
// The check covers the interval [start, end), excluding the end position.
func IntervalInert(info *types.Info, parent inspector.Cursor, absorbedDecls []astutil.NodeIndex, start, end token.Pos) bool {
	// Iterate over all nodes in the parent to find statements in the interval.
	for s := range parent.Preorder(
		// keep-sorted start
		(*ast.AssignStmt)(nil),
		(*ast.BranchStmt)(nil),
		(*ast.CaseClause)(nil),
		(*ast.CommClause)(nil),
		(*ast.DeferStmt)(nil),
		(*ast.ExprStmt)(nil),
		(*ast.ForStmt)(nil),
		(*ast.GenDecl)(nil),
		(*ast.GoStmt)(nil),
		(*ast.IfStmt)(nil),
		(*ast.IncDecStmt)(nil),
		(*ast.LabeledStmt)(nil),
		(*ast.RangeStmt)(nil),
		(*ast.ReturnStmt)(nil),
		(*ast.SelectStmt)(nil),
		(*ast.SendStmt)(nil),
		(*ast.SwitchStmt)(nil),
		(*ast.TypeSwitchStmt)(nil),
		// keep-sorted end

		// Note regarding missing ast.Stmt types:
		// - *ast.BlockStmt is covered by its sub-statements
		// - *ast.DeclStmt is covered by *ast.GenDecl
		// - *ast.EmptyStmt has no side effects
	) {
		n := s.Node()

		if n.Pos() >= end {
			break // We've moved past the area of interest
		}

		if n.End() <= start {
			continue // Before the start of the interval
		}

		if idx := astutil.NodeIndexOf(s); slices.Contains(absorbedDecls, idx) {
			continue
		}

		switch stmt := n.(type) {
		case *ast.AssignStmt:
			if inertShortDecl(info, stmt) {
				continue // Safe declaration
			}

		case *ast.GenDecl:
			if inertVarDecl(info, stmt) {
				continue // Safe declaration
			}
		}

		return false // Found a statement with potential side effects
	}

	return true
}

// inertShortDecl analyzes an assignment statement to determine if it declares a
// constant expression without side effects.
//
// It ensures that:
// 1. It is a short variable declaration (:=).
// 2. All identifiers on the LHS are *new* definitions (no reassignments).
// 3. All expressions on the RHS are inert (constants or safe built-ins).
func inertShortDecl(info *types.Info, stmt *ast.AssignStmt) bool {
	if stmt.Tok != token.DEFINE {
		return false
	}

	for _, id := range stmt.Lhs {
		id, ok := id.(*ast.Ident)
		if !ok {
			return false
		}

		if id.Name == "_" {
			continue
		}

		// Ensure the identifier defines a new object.
		// If Defs[id] is nil, it means it's a reassignment of an existing variable,
		// which is a side effect we must avoid.
		if obj, ok := info.Defs[id]; !ok || obj == nil {
			return false
		}
	}

	for _, expr := range stmt.Rhs {
		if !inertExpr(info, expr) {
			return false
		}
	}

	return true
}

// inertVarDecl checks if a GenDecl AST node represents a `var` declaration that includes initialization values.
func inertVarDecl(info *types.Info, stmt *ast.GenDecl) bool {
	if stmt.Tok != token.VAR { // type declaration and const are safe
		return true
	}

	for _, spec := range stmt.Specs {
		// A ValueSpec with values implies execution (initialization).
		if spec, ok := spec.(*ast.ValueSpec); ok {
			for _, expr := range spec.Values {
				// Check for constant
				if !inertExpr(info, expr) {
					return false
				}
			}
		}
	}

	return true
}

// inertExpr determines if an expression has no side effects, such as being a constant or involving `new` with constant arguments.
func inertExpr(info *types.Info, expr ast.Expr) bool {
	if tv, ok := info.Types[expr]; ok && tv.Value != nil {
		return true
	}

	// Check for new(...)
	call, ok := ast.Unparen(expr).(*ast.CallExpr)
	if !ok || !builtin(info, call.Fun) {
		return false
	}

	for _, arg := range call.Args {
		// Check for type or constant argument
		if tv, ok := info.Types[arg]; !ok || !tv.IsType() && tv.Value == nil {
			return false
		}
	}

	return true
}

// builtin checks if the call expression is a call to the built-in `new` function.
func builtin(info *types.Info, fun ast.Expr) bool {
	id, ok := ast.Unparen(fun).(*ast.Ident)
	if !ok || id.Name != "new" && id.Name != "make" {
		return false
	}

	if _, ok := info.Uses[id].(*types.Builtin); !ok {
		return false
	}

	return true
}
