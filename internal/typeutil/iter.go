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

package typeutil

import (
	"go/ast"
	"go/token"
	"iter"
)

// AllConstDecls is an iterator over all constant declarations in the files.
func AllConstDecls(files []*ast.File) iter.Seq[*ast.ValueSpec] {
	return func(yield func(*ast.ValueSpec) bool) {
		iterateOverDecls(files, func(g *ast.GenDecl) bool {
			if g.Tok != token.CONST {
				return true // non-matching declarations
			}

			return iterateOverSpecs(g, yield)
		})
	}
}

// iterateOverDecls iterates over declarations of a given type D.
func iterateOverDecls[D ast.Decl](files []*ast.File, yield func(D) bool) {
	for _, f := range files {
		for _, decl := range f.Decls {
			d, ok := decl.(D)
			if !ok {
				continue
			}

			if !yield(d) {
				return
			}
		}
	}
}

// iterateOverSpecs iterates over specifications of type S in a generic declaration.
func iterateOverSpecs[S ast.Spec](decl *ast.GenDecl, yield func(S) bool) bool {
	for _, spec := range decl.Specs {
		spec, ok := spec.(S)
		if !ok { // should not happen
			continue
		}

		if !yield(spec) {
			return false
		}
	}

	return true
}
