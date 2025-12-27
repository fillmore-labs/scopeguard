// Copyright 2026 Oliver Eikemeier. All Rights Reserved.
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

package report

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
)

// Renamer handles the renaming of shadowed variables by generating unique names.
//
// It ensures uniqueness by checking the variable's scope hierarchy for naming conflicts.
//
// The Renamer uses lazy initialization for its internal maps, only allocating memory
// when the first variable is renamed.
type Renamer struct {
	// renamed tracks variables that have already been processed to prevent duplicate renaming.
	renamed map[*types.Var]struct{}

	// count tracks the number of times a variable name has been used as a prefix for a new name.
	// This ensures deterministic suffix generation (_1, _2, etc.) across multiple renames.
	count map[string]int
}

// NewRenamer creates a new Renamer instance.
// The actual initialization of internal maps is deferred until the first call to [Renamer.Renames].
func NewRenamer() *Renamer {
	return &Renamer{}
}

// Renames generates [analysis.SuggestedFix]s to rename a shadowed variable.
// It ensures the new name is unique within the variable's scope by checking the scope hierarchy.
//
// The method returns nil if no renaming should be done or the variable has already been renamed.
func (r *Renamer) Renames(info *types.Info, fdecl inspector.Cursor, v *types.Var) []analysis.SuggestedFix {
	if r == nil {
		return nil
	}

	// Has this variable already been renamed?
	if _, ok := r.renamed[v]; ok {
		return nil
	}

	name := v.Name()

	suffix, ok := r.uniqueSuffix(v.Parent(), name)
	if !ok {
		return nil
	}

	offset := len(name)

	var edits []analysis.TextEdit

	// Find all occurrences of this variable (both definitions and uses)
	for c := range fdecl.Preorder((*ast.Ident)(nil)) {
		id, ok := c.Node().(*ast.Ident)
		if !ok || !idIsVar(info, id, v) {
			continue
		}

		pos := token.Pos(int(id.NamePos) + offset)
		edits = append(edits, analysis.TextEdit{Pos: pos, NewText: suffix})
	}

	if len(edits) == 0 {
		return nil
	}

	// Mark this variable as renamed to prevent duplicate processing
	if r.renamed == nil {
		r.renamed = map[*types.Var]struct{}{}
	}
	r.renamed[v] = struct{}{}

	return []analysis.SuggestedFix{{Message: "Rename variable " + name, TextEdits: edits}}
}

// idIsVar checks if the given identifier corresponds to the specified variable.
func idIsVar(info *types.Info, id *ast.Ident, v *types.Var) bool {
	if use, ok := info.Uses[id]; ok {
		return use == v
	}

	if def, ok := info.Defs[id]; ok {
		return def == v
	}

	return false
}

// uniqueSuffix generates a deterministic unique suffix for a variable name.
//
// The method checks both parent and child scopes to ensure the new name doesn't
// conflict with any existing variables in the scope hierarchy.
func (r *Renamer) uniqueSuffix(scope *types.Scope, name string) ([]byte, bool) {
	if name == "_" {
		return nil, false
	}

	const maxTries = 99

	c := r.count[name]

	for range maxTries {
		c++
		suffix := "_" + strconv.Itoa(c)

		// Check if this name conflicts with any existing variable in the scope hierarchy
		if fullName := name + suffix; checkParents(scope, fullName) || checkChildren(scope, fullName) {
			continue
		}

		// Found a unique name: persist the counter and return the suffix
		if r.count == nil {
			r.count = make(map[string]int)
		}
		r.count[name] = c

		return []byte(suffix), true
	}

	return nil, false
}

// checkParents checks if the name is already defined in the scope or any of its parent scopes.
func checkParents(scope *types.Scope, name string) bool {
	for parent := scope; parent != nil; parent = parent.Parent() {
		if parent.Lookup(name) != nil {
			return true
		}
	}

	return false
}

// checkChildren recursively checks if the name is defined in any of the child scopes.
//
// This performs a depth-first search through the scope tree. While this could be
// expensive for deeply nested scopes, it's necessary to ensure the renamed variable
// doesn't conflict with any inner scope declarations. In practice, most functions
// have modest nesting depth, making this acceptable.
func checkChildren(scope *types.Scope, name string) bool {
	for child := range scope.Children() {
		if child.Lookup(name) != nil {
			return true
		}

		if checkChildren(child, name) {
			return true
		}
	}

	return false
}
