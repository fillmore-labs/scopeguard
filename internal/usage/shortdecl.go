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

// handleShortDecl processes short variable declarations (:=).
func (c *collector) handleShortDecl(stmt *ast.AssignStmt, decl astutil.NodeIndex) {
	// The scope of a variable identifier declared inside a function begins at the end of the ShortVarDecl.
	assignmentDone := stmt.End()

	// For each identifier on the LHS
	for idx, expr := range stmt.Lhs {
		id, ok := expr.(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}

		// Check for a new declaration
		if _, ok := c.recordDeclaration(id, decl, assignmentDone); ok {
			// Recorded a new variable definition
			continue
		}

		// Otherwise, it must be a reassignment of an existing variable
		v, ok := c.TypesInfo.Uses[id].(*types.Var)
		if !ok {
			astutil.InternalError(c.Pass, id, "Unknown declaration for variable %s", id.Name)
			continue
		}

		declarations := c.declarations[v]
		if len(declarations) == 0 {
			astutil.InternalError(c.Pass, expr, "Untracked redeclaration of %s", id.Name)

			// If the declaration is not tracked, create a placeholder entry.
			declarations = []DeclarationNode{{Decl: astutil.InvalidNode, Usage: UsageUsed}}
		}

		flags := usageFlagsFromAssignedType(v, assignedType(c.TypesInfo, stmt, idx))
		c.declarations[v] = append(declarations, DeclarationNode{Decl: decl, Usage: flags})

		c.current[v] = declUsage{start: assignmentDone, ignore: id.NamePos}

		c.UpdateShadows(v, id, assignmentDone)

		c.TrackNestedAssignment(v, id, assignmentDone, decl)
	}
}

func usageFlagsFromAssignedType(v *types.Var, assignedType types.Type) Flags {
	switch {
	case assignedType == types.Typ[types.UntypedNil]:
		// The predeclared identifier nil cannot be used to initialize a variable with no explicit type.
		// https://go.dev/ref/spec#Variable_declarations
		return UsageUsedAndTypeChange | UsageUntypedNil

	case !types.Identical(v.Type(), assignedType):
		return UsageTypeChange

	default:
		return UsageNone
	}
}

// assignedType finds the inferred type of the assigned variable.
func assignedType(info *types.Info, stmt *ast.AssignStmt, idx int) types.Type {
	switch len(stmt.Rhs) {
	case len(stmt.Lhs):
		expr := stmt.Rhs[idx]

		// This is used because [types.Checker] calls `updateExprType` for untyped constants.
		//
		// Note that this is a simplified implementation that only handles numeric and string literals or
		// identifiers denoting a constant, not all constant expressions.
		switch expr := ast.Unparen(expr).(type) {
		case *ast.BasicLit:
			switch expr.Kind {
			case token.INT:
				return types.Typ[types.Int]
			case token.FLOAT:
				return types.Typ[types.Float64]
			case token.IMAG:
				return types.Typ[types.Complex128]
			case token.CHAR:
				return universeRune
			case token.STRING:
				return types.Typ[types.String]
			}

		case *ast.Ident:
			if obj, ok := info.Uses[expr]; ok {
				return types.Default(obj.Type())
			}
		}

		return info.Types[expr].Type

	case 1:
		if tuple, ok := info.Types[stmt.Rhs[0]].Type.(*types.Tuple); ok {
			return tuple.At(idx).Type()
		}
	}

	return nil
}

// universeRune is the object for the predeclared "rune" type.
var universeRune = types.Universe.Lookup("rune").Type()
