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

// handleDeclStmt processes var declarations (var x, y = ...).
func (c *collector) handleDeclStmt(gen *ast.GenDecl, decl astutil.NodeIndex) {
	for _, spec := range gen.Specs {
		vspec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		// The scope of a declared variable identifier begins at the end of the VarSpec.
		start := vspec.End()
		for _, id := range vspec.Names {
			if id.Name == "_" {
				continue // blank identifier
			}

			if _, ok := c.recordDeclaration(id, decl, start); !ok {
				astutil.InternalError(c.Pass, vspec, "Non-definition of variable %s", id.Name)
			}
		}
	}
}

// handleReceiveStmt processes declarations in select communication clauses (case x := <-ch:).
//
// Precondition: stmt.Tok == token.DEFINE.
func (c *collector) handleReceiveStmt(stmt *ast.AssignStmt, decl astutil.NodeIndex) {
	assignmentDone := stmt.End()

	for id := range astutil.AllAssigned(stmt) {
		// Record a new variable definition
		v, ok := c.recordDeclaration(id, decl, assignmentDone)
		if !ok {
			astutil.InternalError(c.Pass, id, "Unknown declaration for variable %s", id.Name)
			continue
		}

		c.markNonMovable(v, decl)
	}
}

// handleRangeStmt processes declarations in range statements.
//
// Precondition: stmt.Tok == token.DEFINE.
func (c *collector) handleRangeStmt(stmt *ast.RangeStmt, decl astutil.NodeIndex) {
	assignmentDone := stmt.Body.Lbrace
	for _, e := range []ast.Expr{stmt.Key, stmt.Value} {
		if e == nil {
			continue
		}

		id, ok := e.(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}

		// Record a new variable definition
		v, ok := c.recordDeclaration(id, decl, assignmentDone)
		if !ok {
			astutil.InternalError(c.Pass, id, "Unknown range declaration for variable %s", id.Name)
			continue
		}

		c.markNonMovable(v, decl)
	}
}

// handleCaseClause processes implicit declarations in type switch clauses.
func (c *collector) handleCaseClause(clause *ast.CaseClause, decl astutil.NodeIndex) {
	v, ok := c.TypesInfo.Implicits[clause].(*types.Var)
	if !ok {
		return
	}

	// Implicit variable scope starts at the colon of the case clause.
	start := clause.Colon + 1

	c.recordDefinition(v, decl, start)

	c.markNonMovable(v, decl)

	if clause.List != nil {
		c.CheckDeclarationShadowing(c.UsageScope, v, clause.Colon)
	}
}

// recordDeclaration records a new variable definition.
// It initializes the usage tracking with its declaration position.
func (c *collector) recordDeclaration(id *ast.Ident, decl astutil.NodeIndex, start token.Pos) (*types.Var, bool) {
	def, ok := c.TypesInfo.Defs[id]
	if def == nil {
		return nil, ok
	}

	v, ok := def.(*types.Var)
	if !ok {
		astutil.InternalError(c.Pass, id, "Non-variable declaration of %q", id.Name)
		return nil, true
	}

	c.recordDefinition(v, decl, start)

	c.CheckDeclarationShadowing(c.UsageScope, v, v.Pos())

	return v, true
}

func (c *collector) recordDefinition(v *types.Var, decl astutil.NodeIndex, start token.Pos) {
	c.declarations[v] = []DeclarationNode{{Decl: decl, Usage: UsageNone}}
	c.current[v] = declUsage{start: start, ignore: v.Pos()}
}

// markNonMovable marks a variable declaration as non-movable by setting its usage scope to its declaration scope.
func (c *collector) markNonMovable(v *types.Var, decl astutil.NodeIndex) {
	if c.scopeRanges == nil {
		return
	}

	declScope := v.Parent()
	c.scopeRanges[decl] = ScopeRange{Decl: declScope, Usage: declScope} // Not movable
}
