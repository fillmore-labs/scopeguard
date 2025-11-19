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

	"golang.org/x/tools/go/ast/inspector"
)

// findUnused identifies variable declarations that are not used within any recorded scope.
//
// Returns a map where:
//   - Key: nodeIndex of an unused declaration
//   - Value: true if the declaration must be kept for type safety, false if it can be removed
//
// Type safety example:
//
//	var err error                          // First declaration (unused) - must keep
//	a, err := func() (int, MyError) {}()   // Redeclaration: assigns MyError to error (valid)
//	b, err := func() (int, error) {}()     // Redeclaration: assigns error to error (valid)
//
// If we remove the first declaration, err would get type MyError, and the second
// redeclaration might fail type checking (error may not be assignable to MyError).
func (p pass) findUnused(in *inspector.Inspector, decls declResult, scopes map[nodeIndex]scopeRange) map[nodeIndex]bool {
	unused := make(map[nodeIndex]bool)

	// Identify unused declarations and determine which can be safely removed
	for v, declIndices := range decls.decls {
		for i, declIndex := range declIndices {
			if _, ok := scopes[declIndex]; ok {
				continue // Declaration has usage
			}

			if _, notMovable := decls.notMovable[declIndex]; notMovable {
				continue // Named return redeclaration - cannot be removed
			}

			// Determine if this unused declaration must be kept for type safety
			keep := false

			if i == 0 && len(declIndices) > 1 {
				// First declaration with redeclarations: check if removal would break type compatibility
				// We look for a redeclaration that hasn't been moved (stays in original scope)
				// because moving changes code structure and may affect type inference differently
				for _, index := range declIndices[1:] {
					if usage, ok := scopes[index]; ok && usage.decl == usage.usage {
						// Found a non-moved redeclaration
						node := in.At(index).Node()

						// Check to detect type differences
						if rhsType := p.getDeclType(v, node); rhsType == nil || !types.Identical(v.Type(), rhsType) {
							// Types are incompatible - removing the first declaration would change the variable's type
							keep = true
						}

						break
					}
				}
			}

			// Record this unused declaration
			// If multiple variables share this declaration node (e.g., var x, y int),
			// keep it if ANY variable needs it for type safety (OR logic)
			unused[declIndex] = unused[declIndex] || keep
		}
	}

	return unused
}

// getDeclType extracts the type that would be assigned to a variable at a specific redeclaration point.
//
// For short variable declarations (`:=`), it determines what type the variable would receive
// from the RHS expression(s). This is used to check type compatibility when considering removal
// of unused first declarations.
//
// Examples:
//
//	x, y := 1, 2                    // Simple: y would get type int
//	x, y := foo()                   // Multi-value: y would get type of second return value
//
// Returns nil if the type cannot be determined or if the node is not a short variable declaration.
func (p pass) getDeclType(v *types.Var, node ast.Node) types.Type {
	stmt, ok := node.(*ast.AssignStmt)
	if !ok || stmt.Tok != token.DEFINE {
		return nil
	}

	// Find which position on the LHS this variable occupies
	pos := namePosition(stmt.Lhs, v.Name())
	if pos < 0 {
		return nil
	}

	// Extract the RHS type that would be assigned to this variable
	switch len(stmt.Rhs) {
	case len(stmt.Lhs): // Simple case: one RHS expression per LHS variable
		expr := stmt.Rhs[pos]
		if typ, ok := p.pass.TypesInfo.Types[expr]; ok {
			return typ.Type
		}

	case 1: // Multi-value case: one RHS expression returning multiple values (e.g., function call)
		expr := stmt.Rhs[0]

		typ, ok := p.pass.TypesInfo.Types[expr]
		if !ok {
			break
		}

		// Extract the type from the tuple of return values
		tuple, ok := typ.Type.(*types.Tuple)
		if !ok || pos >= tuple.Len() {
			break
		}

		return tuple.At(pos).Type()
	}

	return nil
}

// namePosition finds the index of a variable name in the LHS of an assignment statement.
// Returns -1 if the name is not found.
func namePosition(lhs []ast.Expr, name string) int {
	for i, expr := range lhs {
		if id, ok := expr.(*ast.Ident); ok && id.Name == name {
			return i
		}
	}

	return -1
}
