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
	"context"
	"fmt"
	"go/ast"
	"go/types"
	"runtime/trace"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"

	"fillmore-labs.com/scopeguard/internal/astutil"
	"fillmore-labs.com/scopeguard/internal/category"
	"fillmore-labs.com/scopeguard/internal/usage/check"
)

// reportUsedAfterShadow emits diagnostics for variables used after previously shadowed.
func reportUsedAfterShadow(ctx context.Context, p *analysis.Pass, currentFile astutil.CurrentFile, fdecl inspector.Cursor, shadows []check.ShadowUse, renames map[string][]string, rename bool) {
	if len(shadows) == 0 {
		return
	}

	defer trace.StartRegion(ctx, "ReportShadowed").End()

	var renamer *Renamer
	if rename || len(renames) > 0 {
		renamer = NewRenamer(renames, rename)
	}

	in := fdecl.Inspector()

	for _, shadowed := range shadows {
		use := shadowed.Use.Node(in)
		if currentFile.NoLintComment(use.Pos()) {
			continue
		}

		suggestedFixes := renamer.Renames(p.TypesInfo, fdecl, shadowed.Var)
		cat := category.Shadowed

		p.Report(analysis.Diagnostic{
			Pos:            use.Pos(),
			End:            use.End(),
			Category:       cat,
			Message:        fmt.Sprintf("Variable '%s' used after previously shadowed (sg:%s)", shadowed.Var.Name(), cat),
			Related:        []analysis.RelatedInformation{{Pos: shadowed.ShadowPos, Message: "After this declaration"}},
			SuggestedFixes: suggestedFixes,
		})
	}
}

// Renamer handles the renaming of shadowed variables by generating unique names.
//
// It ensures uniqueness by checking the variable's scope hierarchy for naming conflicts.
type Renamer struct {
	// processed tracks variables that have already been processed to prevent duplicate renaming.
	processed map[*types.Var]struct{}

	// count tracks the number of times a variable name has been used as a prefix for a new name.
	// This ensures deterministic suffix generation (_1, _2, etc.) across multiple renames.
	count map[string]int

	renames map[string][]string
	rename  bool
}

// NewRenamer creates a new Renamer instance.
// The actual initialization of internal maps is deferred until the first call to [Renamer.Renames].
func NewRenamer(renames map[string][]string, rename bool) *Renamer {
	return &Renamer{
		processed: make(map[*types.Var]struct{}),
		count:     make(map[string]int),
		renames:   renames,
		rename:    rename,
	}
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
	if _, ok := r.processed[v]; ok {
		return nil
	}

	// Mark this variable as renamed to prevent duplicate processing
	r.processed[v] = struct{}{}

	name, parent := v.Name(), v.Parent()

	newName, ok := r.uniqueName(parent, name)
	if !ok {
		return nil
	}

	scope, ok := fdecl.FindByPos(parent.Pos(), parent.End())
	if !ok {
		return nil
	}

	newText := []byte(newName)

	var edits []analysis.TextEdit

	foundDef := false
	// Find all occurrences of this variable (both definitions and uses)
	for c := range scope.Preorder((*ast.Ident)(nil)) {
		id := c.Node().(*ast.Ident)

		def, ok := idIsVar(info, id, v)
		if !ok {
			continue
		}

		if def {
			foundDef = true
		}

		edits = append(edits, analysis.TextEdit{Pos: id.Pos(), End: id.End(), NewText: newText})
	}

	// Avoid rename of implicit variables
	if !foundDef {
		return nil
	}

	msg := fmt.Sprintf("Rename '%s' to '%s'", name, newName)

	return []analysis.SuggestedFix{{Message: msg, TextEdits: edits}}
}

// idIsVar checks if the given identifier corresponds to the specified variable.
func idIsVar(info *types.Info, id *ast.Ident, v *types.Var) (def, ok bool) {
	if use, ok := info.Uses[id]; ok {
		return false, use == v
	}

	if obj, ok := info.Defs[id]; ok {
		return true, obj == v
	}

	return false, false
}

// uniqueName generates a deterministic unique suffix for a variable name.
//
// The method checks both parent and child scopes to ensure the new name doesn't
// conflict with any existing variables in the scope hierarchy.
func (r *Renamer) uniqueName(scope *types.Scope, name string) (string, bool) {
	if name == "_" {
		return "", false
	}

	c := r.count[name]
	if names, ok := r.renames[name]; ok {
		if c >= len(names) {
			return "", false
		}

		newName := names[c]
		r.count[name]++

		if newName == "" {
			return "", false
		}

		// Check if this name conflicts with any existing variable in the scope hierarchy
		if checkScopes(scope, newName) {
			return "", false
		}

		return newName, true
	}

	if !r.rename {
		return "", false
	}

	const maxTries = 99
	for range maxTries {
		c++
		newName := name + "_" + strconv.Itoa(c)

		// Check if this name conflicts with any existing variable in the scope hierarchy
		if checkScopes(scope, newName) {
			continue
		}

		// Found a unique name: persist the counter and return the suffix
		r.count[name] = c

		return newName, true
	}

	return "", false
}

// checkScopes checks if the name is already defined in the scope or any of its parent or child scopes.
func checkScopes(scope *types.Scope, name string) bool {
	// check if the name is already defined in the scope or any of its parent scopes.
	for parent := scope; parent != nil; parent = parent.Parent() {
		if parent.Lookup(name) != nil {
			return true
		}
	}

	// check child scopes.
	return checkChildren(scope, name)
}

// checkChildren recursively checks if the name is defined in any of the child scopes.
func checkChildren(scope *types.Scope, name string) bool {
	for child := range scope.Children() {
		if child.Lookup(name) != nil {
			return true
		}

		// This performs a depth-first search through the scope tree. While this could be
		// expensive for deeply nested scopes, it's necessary to ensure the renamed variable
		// doesn't conflict with any inner scope declarations. In practice, most functions
		// have modest nesting depth, making this acceptable.
		if checkChildren(child, name) {
			return true
		}
	}

	return false
}
