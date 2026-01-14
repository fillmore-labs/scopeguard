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
	"go/types"
)

// _knownFuncs are functions that do not return.
var _knownFuncs = map[FuncName]struct{}{
	{Path: "log", Name: "Fatal"}:    {},
	{Path: "log", Name: "Fatalf"}:   {},
	{Path: "os", Name: "Exit"}:      {},
	{Path: "syscall", Name: "Exit"}: {},
}

// canReturn iteratively unwraps an expression to find the underlying function declaration.
func canReturn(info *types.Info, n *ast.CallExpr) bool {
	for ex := n.Fun; ; {
		switch e := ex.(type) {
		case *ast.Ident:
			uses := info.Uses[e]
			if uses == panicBuiltin {
				return false
			}

			fun, ok := uses.(*types.Func)
			if !ok {
				return true
			}

			name := FuncNameOf(fun)
			_, ok = _knownFuncs[name]

			return !ok

		case *ast.SelectorExpr:
			// e.Sel is an identifier qualified by e.X
			fun, ok := info.Uses[e.Sel].(*types.Func)
			if !ok {
				return true
			}

			name := FuncNameOf(fun)
			_, ok = _knownFuncs[name]

			return !ok

		case *ast.IndexExpr: // Generic function instantiation with a type parameter ("myFunc[T]").
			ex = e.X // Unwrap to the function identifier.

		case *ast.IndexListExpr: // Generic function instantiation with multiple type parameters ("myFunc[T, U]").
			ex = e.X // Unwrap to the function identifier.

		case *ast.ParenExpr: // Parenthesized expression ("(myFunc)")
			ex = e.X // Unwrap to the inner expression.

		default: // Function variable, pointer, or other non-declarative function reference.
			return true
		}
	}
}

var panicBuiltin = types.Universe.Lookup("panic").(*types.Builtin)
