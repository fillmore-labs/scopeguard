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

package typeutil

import (
	"go/ast"
	"go/types"
)

// FuncOf iteratively unwraps an expression to find the underlying function call.
func FuncOf(info *types.Info, call *ast.CallExpr) types.Object {
	ex := call.Fun
	typeParams := false

unwarp:
	switch e := ex.(type) {
	case *ast.Ident:
		return info.Uses[e]

	case *ast.SelectorExpr:
		return info.Uses[e.Sel]

	case *ast.IndexExpr: // Generic function instantiation with a type parameter ("myFunc[T]").
		if typeParams { // should not happen, duplicate type parameters
			return nil
		}

		if !checkTypeParameters(info, []ast.Expr{e.Index}) {
			return nil // Not a type, but an array/slice index.
		}

		typeParams = true
		ex = e.X // Unwrap to the function identifier.

		goto unwarp

	case *ast.IndexListExpr: // Generic function instantiation with multiple type parameters ("myFunc[T, U]").
		if typeParams { // should not happen, duplicate type parameters
			return nil
		}

		if !checkTypeParameters(info, e.Indices) { // should not happen
			return nil
		}

		typeParams = true
		ex = e.X // Unwrap to the function identifier.

		goto unwarp

	case *ast.ParenExpr: // Parenthesized expression ("(myFunc)")
		ex = e.X // Unwrap to the inner expression.
		goto unwarp

	default: // Function variable, pointer, or another non-declarative function reference.
		return nil
	}
}

// checkTypeParameters validates type parameters from an IndexExpr
// or IndexListExpr. It uses the provided types.Info to verify that each
// expression represents a type.
//
// If any expression is not a type, it returns false.
func checkTypeParameters(info *types.Info, indices []ast.Expr) bool {
	for _, index := range indices {
		if !info.Types[index].IsType() { // Must be a type parameter, not an array/slice index.
			return false
		}
	}

	return true
}
