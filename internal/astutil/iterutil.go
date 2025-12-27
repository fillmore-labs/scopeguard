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

package astutil

import (
	"go/ast"
	"go/token"
	"iter"
)

// AllAssignedNames yields all assigned variable names.
func AllAssignedNames(stmt *ast.AssignStmt) iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, expr := range stmt.Lhs {
			id, ok := expr.(*ast.Ident)
			if !ok || id.Name == "_" {
				continue // blank identifier
			}

			if !yield(id.Name) {
				return
			}
		}
	}
}

// AllDeclaredNames yields all declared variable names.
func AllDeclaredNames(stmt *ast.DeclStmt) iter.Seq[string] {
	decl, ok := stmt.Decl.(*ast.GenDecl)
	if !ok || decl.Tok != token.VAR {
		return func(func(string) bool) {}
	}

	return func(yield func(string) bool) {
		for _, spec := range decl.Specs {
			vspec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for _, id := range vspec.Names {
				if id.Name == "_" {
					continue // blank identifier
				}

				if !yield(id.Name) {
					return
				}
			}
		}
	}
}
