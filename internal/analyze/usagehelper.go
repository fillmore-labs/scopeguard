// Copyright 2025 Oliver Eikemeier. All Rights Reserved.
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
)

// hasNamedResults reports whether the function has named result parameters.
func hasNamedResults(results *ast.FieldList) bool {
	return results != nil && len(results.List) > 0 && len(results.List[0].Names) > 0
}

// exprType finds the inferred type of the assigned variable.
func exprType(info *types.Info, stmt *ast.AssignStmt, idx int) types.Type {
	switch len(stmt.Rhs) {
	case len(stmt.Lhs):
		// [types.Checker] calls `updateExprType` for untyped constants.
		return assignedType(info, stmt.Rhs[idx])

	case 1:
		if tuple, ok := info.Types[stmt.Rhs[0]].Type.(*types.Tuple); ok {
			return tuple.At(idx).Type()
		}
	}

	return nil
}

// universeRune is the object for the predeclared "rune" type.
var universeRune = types.Universe.Lookup("rune")

// assignedType returns the type of the expressions.
//
// Note that this is a simplified implementation that only handles numeric and string literals or
// identifiers denoting a constant, not all constant expressions.
func assignedType(info *types.Info, expr ast.Expr) types.Type {
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
			return universeRune.Type()
		case token.STRING:
			return types.Typ[types.String]
		}

	case *ast.Ident:
		if obj, ok := info.Uses[expr]; ok {
			return types.Default(obj.Type())
		}
	}

	return info.Types[expr].Type
}
