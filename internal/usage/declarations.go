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

	"golang.org/x/tools/go/ast/inspector"

	"fillmore-labs.com/scopeguard/internal/astutil"
)

// handleFunc processes function parameters and results, recording their declarations.
func (c *collector) handleFunc(body inspector.Cursor, recv *ast.FieldList, typ *ast.FuncType) {
	start, decl := body.Node().Pos(), astutil.NodeIndexOf(body.Parent())

	for _, list := range [...]*ast.FieldList{recv, typ.Params, typ.Results} {
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
				c.usages[v] = []NodeUsage{{Decl: decl, Usage: UsageUsed}}

				// v.Parent() == uc.TypesInfo.Scopes[typ]
				c.notMovable(decl, v)
			}
		}
	}
}

// handleDeclStmt processes var declarations (var x, y = ...).
func (c *collector) handleDeclStmt(gen *ast.GenDecl, decl astutil.NodeIndex) {
	for _, spec := range gen.Specs {
		vspec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		// The scope of a variable identifier declared inside a function begins at the end of the VarSpec.
		c.recordDeclarations(vspec, decl)
	}
}

// recordDeclarations records all variable definitions from a single ValueSpec.
func (c *collector) recordDeclarations(vspec *ast.ValueSpec, decl astutil.NodeIndex) {
	start := vspec.End()
	for _, id := range vspec.Names {
		if id.Name == "_" {
			continue // blank identifier
		}

		def, ok := c.TypesInfo.Defs[id]
		if !ok {
			astutil.InternalError(c.Pass, vspec, "Non-definition of variable %s", id.Name)
			continue
		}

		c.recordDeclaration(decl, start, id, def)
	}
}

// handleReceiveStmt processes assignments in select communication clauses (case x := <-ch:).
func (c *collector) handleReceiveStmt(stmt *ast.AssignStmt, decl astutil.NodeIndex) {
	assignmentDone := stmt.End()

	for _, id := range stmt.Lhs {
		id, ok := id.(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}

		v, ok := c.TypesInfo.Defs[id].(*types.Var)
		if !ok {
			astutil.InternalError(c.Pass, id, "Unknown declaration for variable %s", id.Name)
			continue
		}

		// Record a new variable definition
		c.recordDeclaration(decl, assignmentDone, id, v)

		c.notMovable(decl, v)
	}
}

func (c *collector) handleRangeStmt(stmt *ast.RangeStmt, idx astutil.NodeIndex) {
	assignmentDone := stmt.Body.Lbrace
	for _, e := range []ast.Expr{stmt.Key, stmt.Value} {
		if e == nil {
			continue
		}

		id, ok := ast.Unparen(e).(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}

		if def, ok := c.TypesInfo.Defs[id]; ok {
			// Record a new variable definition
			c.recordDeclaration(idx, assignmentDone, id, def)

			continue
		}

		astutil.InternalError(c.Pass, id, "Unknown range declaration for variable %s", id.Name)
	}
}

// recordDeclaration records a new variable definition from a short declaration.
// It initializes the usage tracking for this variable with its declaration position.
func (c *collector) recordDeclaration(decl astutil.NodeIndex, start token.Pos, id *ast.Ident, def types.Object) {
	v, ok := def.(*types.Var)
	if !ok {
		astutil.InternalError(c.Pass, id, "Non-variable declaration of %q", id.Name)
		return
	}

	// Variable declaration
	if _, ok := c.usages[v]; ok {
		astutil.InternalError(c.Pass, id, "Redeclaration of variable %q", id.Name)
	}

	usage := NodeUsage{Decl: decl, Usage: UsageNone}
	c.usages[v] = []NodeUsage{usage}

	c.current[v] = declUsage{start: start, ignore: id.NamePos}

	c.RecordShadowingDeclaration(c.UsageScope, v, id, decl)
}

// notMovable marks a variable declaration as non-movable by setting its usage scope to its declaration scope.
func (c *collector) notMovable(decl astutil.NodeIndex, v *types.Var) {
	if c.scopeRanges == nil {
		return
	}

	declScope := v.Parent()
	c.scopeRanges[decl] = ScopeRange{Decl: declScope, Usage: declScope} // Not movable
}
