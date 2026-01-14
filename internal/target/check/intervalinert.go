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
		(*ast.BadStmt)(nil),
		(*ast.BranchStmt)(nil),
		(*ast.CaseClause)(nil),
		(*ast.DeferStmt)(nil),
		(*ast.ExprStmt)(nil),
		(*ast.ForStmt)(nil),
		(*ast.DeclStmt)(nil),
		(*ast.GoStmt)(nil),
		(*ast.IfStmt)(nil),
		(*ast.IncDecStmt)(nil),
		(*ast.LabeledStmt)(nil),
		(*ast.RangeStmt)(nil),
		(*ast.ReturnStmt)(nil),
		(*ast.SendStmt)(nil),
		// keep-sorted end

		// Note regarding missing [ast.Stmt] types:
		// - *[ast.BlockStmt] is covered by its [ast.BlockStmt.List] sub-statements
		// - *[ast.CommClause] is covered by [ast.CommClause.Comm] and [ast.CommClause.Body]
		// - *[ast.EmptyStmt] has no side effects
		// - *[ast.SelectStmt] is covered by [ast.SelectStmt.Body]
		// - *[ast.SwitchStmt] is covered by [ast.SwitchStmt.Init] and [ast.SwitchStmt.Body]
		// - *[ast.TypeSwitchStmt] is covered by [ast.TypeSwitchStmt.Init] and [ast.TypeSwitchStmt.Body]
	) {
		n := s.Node()

		if n.Pos() >= end {
			break // We've moved past the area of interest
		}

		if n.Pos() < start {
			continue // Before the start of the interval
		}

		if idx := astutil.NodeIndexOf(s); slices.Contains(absorbedDecls, idx) {
			continue
		}

		if InertStmt(info, n) {
			continue // Safe
		}

		return false // Found a statement with potential side effects
	}

	return true
}

// InertStmt determines if a statement is "inert," meaning it has no side effects or observable interactions.
func InertStmt(info *types.Info, node ast.Node) bool {
	switch stmt := node.(type) {
	case *ast.AssignStmt:
		return inertShortDecl(info, stmt)

	case *ast.DeclStmt:
		return inertVarDecl(info, stmt)

	case *ast.EmptyStmt:
		return true

	default:
		return false
	}
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

	for id := range astutil.AllAssigned(stmt) {
		// Ensure the identifier defines a new object.
		// If its not in Defs[id], it means it's a reassignment of an existing variable,
		// which is a side effect we must avoid.
		if _, ok := info.Defs[id]; !ok {
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
func inertVarDecl(info *types.Info, stmt *ast.DeclStmt) bool {
	decl := stmt.Decl.(*ast.GenDecl)
	if decl.Tok != token.VAR { // type declaration and const are safe
		return true
	}

	for _, spec := range decl.Specs {
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
	// Check for type or constant argument
	if tv, ok := info.Types[expr]; ok && (tv.IsType() || tv.Value != nil) {
		return true
	}

unpack:
	switch ex := expr.(type) {
	case *ast.ParenExpr:
		expr = ex.X
		goto unpack

	case *ast.CallExpr:
		// Check for new(...), make(...)
		if !builtin(info, ex.Fun) {
			return false
		}

		for _, arg := range ex.Args {
			if !inertExpr(info, arg) {
				return false
			}
		}

		return true

	case *ast.UnaryExpr:
		expr = ex.X
		goto unpack

	case *ast.CompositeLit:
		for _, e := range ex.Elts {
			if !inertExpr(info, e) {
				return false
			}
		}

		return true

	default:
		return false
	}
}

// builtin checks if the call expression is a call to the built-in `new` or `make` function.
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
