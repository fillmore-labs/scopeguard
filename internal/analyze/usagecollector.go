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

package analyze

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ast/edge"
	"golang.org/x/tools/go/ast/inspector"
)

// usageCollector collects variable declarations and tracks their usages within a function body.
type usageCollector struct {
	// Pass is an embedded pass for type information and error reporting
	Pass

	// ScopeIndex is an embedded scope analyzer for scope hierarchy navigation
	ScopeIndex

	// ShadowChecker is an embedded shadow checker.
	ShadowChecker

	// NestedChecker is an embedded checker for nested assignments.
	NestedChecker

	// scopeRanges maps declaration indices to their scope ranges (declaration scope + usage scope).
	scopeRanges map[NodeIndex]ScopeRange

	// usages maps variables to their usages history.
	// The first entry is typically the initial declaration; subsequent entries are reassignments.
	usages map[*types.Var][]NodeUsage

	// current maps variables to their current (re)declaration.
	current map[*types.Var]declUsage
}

// declUsage tracks the scope and position of a variable's last declaration.
type declUsage struct {
	// start is the position where the variable's scope begins.
	// For short variable declarations, this is the end of the statement.
	start token.Pos

	// ignore is the position of the declaration identifier itself.
	// Usages at this position (LHS of assignment) are ignored to avoid
	// counting the declaration as a usage.
	ignore token.Pos
}

// result returns the collected usage information.
func (uc *usageCollector) result() (UsageData, UsageDiagnostics) {
	return UsageData{
			ScopeRanges: uc.scopeRanges,
			Usages:      uc.usages,
		}, UsageDiagnostics{
			Shadows: uc.UsedAfterShadow(),
			Nested:  uc.NestedAssigned(),
		}
}

// inspectBody traverses the AST of a function body to collect:
//   - Short variable declarations (x :=)
//   - Var declarations (var x int)
//   - Variable usages
//
// For each declaration, it tracks the tightest scope containing all usages,
// which determines if the declaration can be moved to a narrower scope.
func (uc *usageCollector) inspectBody(body inspector.Cursor, recv *ast.FieldList, typ *ast.FuncType) {
	uc.handleFunc(body, recv, typ)

	types := []ast.Node{
		// keep-sorted start
		(*ast.AssignStmt)(nil),
		(*ast.DeclStmt)(nil),
		(*ast.FuncLit)(nil),
		(*ast.Ident)(nil),
		(*ast.RangeStmt)(nil),
		// keep-sorted end
	}

	if hasNamedResults(typ.Results) {
		// We only need to check return statements for named results.
		types = append(types, (*ast.ReturnStmt)(nil))
	}

	body.Inspect(types, func(c inspector.Cursor) bool {
		switch n := c.Node().(type) {
		// keep-sorted start newline_separated=yes
		case *ast.AssignStmt:
			switch n.Tok {
			case token.ASSIGN:
				vars := extractVars(uc.TypesInfo.Uses, n.Lhs)
				uc.handleAssignedVars(c.Index(), n.Pos(), n.End(), vars)

			case token.DEFINE:
				switch kind, _ := c.ParentEdge(); kind {
				case edge.CommClause_Comm: // Don't consider short declarations in select cases
					uc.handleReceiveStmt(c.Index(), n)
					return true

				case edge.TypeSwitchStmt_Assign: // Don't consider short declarations in type switches
					return true
				}

				uc.handleShortDecl(c.Index(), n)
			}

		case *ast.DeclStmt:
			gen, ok := n.Decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				break
			}

			uc.handleDeclStmt(c.Index(), gen)

		case *ast.FuncLit:
			fbody, ftype := c.ChildAt(edge.FuncLit_Body, -1), n.Type

			// Traverse recursively with different return values
			uc.inspectBody(fbody, nil, ftype)

			return false // Visited recursively in inspectBody, do not descend

		case *ast.Ident:
			if n.Name == "_" {
				break
			}

			uc.handleIdent(c.Index(), n)

		case *ast.RangeStmt:
			if n.Key == nil {
				break
			}

			switch n.Tok {
			case token.ASSIGN:
				vars := extractVars(uc.TypesInfo.Uses, []ast.Expr{n.Key, n.Value})
				uc.handleAssignedVars(c.Index(), n.Pos(), n.Body.Pos(), vars)

			case token.DEFINE:
				uc.handleRangeStmt(c.Index(), n)
			}

		case *ast.ReturnStmt:
			if len(n.Results) > 0 {
				break
			}

			uc.handleNamedResults(c.Index(), typ.Results, n.Pos())

			// keep-sorted end
		}

		return true
	})
}

// handleAssignedVars handles nested assignments and updates shadow tracking when variables are assigned.
func (uc *usageCollector) handleAssignedVars(asgn NodeIndex, pos, assignmentDone token.Pos, vars []assignedVar) {
	for _, vid := range vars {
		uc.UpdateShadows(pos, assignmentDone, vid)

		uc.TrackAssignment(asgn, pos, assignmentDone, vid)
	}
}
