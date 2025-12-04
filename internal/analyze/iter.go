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
	"iter"
)

// allListed yields all listed [*ast.Ident] nodes.
func allListed(list *ast.FieldList) iter.Seq[*ast.Ident] {
	return func(yield func(*ast.Ident) bool) {
		if list == nil {
			return
		}

		for _, names := range list.List {
			for _, id := range names.Names {
				if id.Name == "_" {
					continue // blank identifier
				}

				if !yield(id) {
					return
				}
			}
		}
	}
}

// allAssigned yields all assigned [*ast.Ident] nodes.
func allAssigned(stmt *ast.AssignStmt) iter.Seq[*ast.Ident] {
	return func(yield func(*ast.Ident) bool) {
		for _, expr := range stmt.Lhs {
			id, ok := expr.(*ast.Ident)
			if !ok || id.Name == "_" {
				continue // blank identifier
			}

			if !yield(id) {
				return
			}
		}
	}
}

// allDeclared yields all declared [*ast.Ident] nodes.
func allDeclared(stmt *ast.DeclStmt) iter.Seq[*ast.Ident] {
	decl, ok := stmt.Decl.(*ast.GenDecl)
	if !ok || decl.Tok != token.VAR {
		return func(func(*ast.Ident) bool) {}
	}

	return func(yield func(*ast.Ident) bool) {
		for _, spec := range decl.Specs {
			vspec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for _, id := range vspec.Names {
				if id.Name == "_" {
					continue // blank identifier
				}

				if !yield(id) {
					return
				}
			}
		}
	}
}
