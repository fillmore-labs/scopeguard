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
	"context"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/edge"
	"golang.org/x/tools/go/ast/inspector"

	"fillmore-labs.com/scopeguard/internal/astutil"
	"fillmore-labs.com/scopeguard/internal/reachability"
	"fillmore-labs.com/scopeguard/internal/scope"
	"fillmore-labs.com/scopeguard/internal/usage/check"
)

// collector collects variable declarations and tracks their usages within a function body.
type collector struct {
	// Pass is an embedded [analysis.Pass] for type information and error reporting
	*analysis.Pass

	// UsageScope is an embedded scope analyzer for scope hierarchy navigation
	scope.UsageScope

	// ShadowChecker is an embedded shadow checker.
	check.ShadowChecker

	// NestedChecker is an embedded checker for nested assignments.
	check.NestedChecker

	// scopeRanges maps declaration indices to their scope ranges (declaration scope + usage scope).
	scopeRanges map[astutil.NodeIndex]ScopeRange

	// declarations maps variables to their declaration history.
	// The first entry is typically the initial declaration; subsequent entries are reassignments.
	declarations map[*types.Var][]DeclarationNode

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
func (c *collector) result() (Result, Diagnostics) {
	return Result{
			scopeRanges:  c.scopeRanges,
			declarations: c.declarations,
		}, Diagnostics{
			Shadows: c.UsedAfterShadow(),
			Nested:  c.NestedAssigned(),
		}
}

// handleFunc traverses the AST of a function body to collect:
//   - Short variable declarations (x :=)
//   - Var declarations (var x int)
//   - Variable usages
//
// For each declaration, it tracks the tightest scope containing all usages,
// which determines if the declaration can be moved to a narrower scope.
func (c *collector) handleFunc(ctx context.Context, body inspector.Cursor, recv *ast.FieldList, fun *ast.FuncType) {
	bodyNode := body.Node().(*ast.BlockStmt)

	oldgraph := c.Graph
	if c.ShadowCheckerEnabled() {
		c.Graph = reachability.NewGraph(ctx, c.TypesInfo, recv, fun, bodyNode, true)
	}

	// Processes function parameters and results, recording their declarations.
	params, results := fun.Params, fun.Results

	decl, start := astutil.NodeIndexOf(body.Parent()), bodyNode.Pos()

	for _, list := range [...]*ast.FieldList{recv, params, results} {
		if list == nil {
			continue
		}

		for _, names := range list.List {
			for _, id := range names.Names {
				if id.Name == "_" {
					continue // blank identifier
				}

				v, ok := c.TypesInfo.Defs[id].(*types.Var)
				if !ok {
					continue
				}

				// Parameter / result declaration
				c.current[v] = declUsage{start: start, ignore: id.NamePos}
				c.declarations[v] = []DeclarationNode{{Decl: decl, Usage: UsageUsed}}
			}
		}
	}

	if c.scopeRanges != nil {
		funScope := c.TypesInfo.Scopes[fun]
		c.scopeRanges[decl] = ScopeRange{Decl: funScope, Usage: funScope} // Not movable
	}

	// Does the function have named result parameters?
	namedResults := results != nil && len(results.List) > 0 && len(results.List[0].Names) > 0

	nodes := []ast.Node{
		// keep-sorted start
		(*ast.AssignStmt)(nil),
		(*ast.CaseClause)(nil),
		(*ast.DeclStmt)(nil),
		(*ast.FuncLit)(nil),
		(*ast.Ident)(nil),
		(*ast.RangeStmt)(nil),
		(*ast.ReturnStmt)(nil),
		// keep-sorted end
	}

	body.Inspect(nodes, func(i inspector.Cursor) bool {
		switch idx := astutil.NodeIndexOf(i); n := i.Node().(type) {
		// keep-sorted start newline_separated=yes
		case *ast.AssignStmt:
			switch n.Tok {
			case token.ASSIGN:
				c.handleAssignedVars(n.Lhs, n.End(), idx)

			case token.DEFINE:
				switch kind, _ := i.ParentEdge(); kind {
				case edge.CommClause_Comm:
					c.handleReceiveStmt(n, idx)
					return true

				case edge.TypeSwitchStmt_Assign: // Don't consider short declarations in type switches
					return true
				}

				c.handleShortDecl(n, idx)
			}

		case *ast.CaseClause:
			c.handleCaseClause(n, idx)

		case *ast.DeclStmt:
			gen, ok := n.Decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				break
			}

			c.handleDeclStmt(gen, idx)

		case *ast.FuncLit:
			fbody, ftype := i.ChildAt(edge.FuncLit_Body, -1), n.Type

			// Traverse recursively with different return values
			c.handleFunc(ctx, fbody, nil, ftype)

			return false // Visited recursively in inspectBody, do not descend

		case *ast.Ident:
			if n.Name == "_" {
				break
			}

			c.handleIdent(n, idx)

		case *ast.RangeStmt:
			if n.Key == nil {
				break
			}

			switch n.Tok {
			case token.ASSIGN:
				c.handleAssignedVars([]ast.Expr{n.Key, n.Value}, n.Body.Pos(), idx)

			case token.DEFINE:
				c.handleRangeStmt(n, idx)
			}

		case *ast.ReturnStmt:
			if !namedResults || len(n.Results) > 0 {
				break
			}

			// Bare return of named results, mark them as used
			c.handleNamedResults(idx, results, n.Pos())

			// keep-sorted end
		}

		return true
	})

	c.Graph = oldgraph
}
