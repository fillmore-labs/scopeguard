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

	"golang.org/x/tools/go/ast/edge"
	"golang.org/x/tools/go/ast/inspector"
)

// usageCollector collects variable declarations and tracks their usages within a function body.
type usageCollector struct {
	pass          // Embedded pass for type information and error reporting
	scopeAnalyzer // Embedded scope analyzer for scope hierarchy navigation

	// scopeRanges maps declaration indices to their scope ranges (declaration scope + usage scope).
	scopeRanges map[nodeIndex]scopeRange

	// decls maps variables to their current declaration information.
	decls map[*types.Var]declUsage

	// usages maps variables to their usages history.
	// The first entry is typically the initial declaration; subsequent entries are reassignments.
	usages map[*types.Var][]nodeUsage
}

// declUsage tracks the scope and position of a variables last declaration.
type declUsage = struct {
	// start is the position where the variable's scope begins.
	// For short variable declarations, this is the end of the statement.
	start token.Pos

	// ignore is the position of the declaration identifier itself.
	// Usages at this position (LHS of assignment) are ignored.
	ignore token.Pos
}

// newUsageCollector creates a new usage collector for analyzing a function body.
func (p pass) newUsageCollector(scopes scopeAnalyzer) usageCollector {
	return usageCollector{
		pass:          p,
		scopeAnalyzer: scopes,
		scopeRanges:   make(map[nodeIndex]scopeRange),
		decls:         make(map[*types.Var]declUsage),
		usages:        make(map[*types.Var][]nodeUsage),
	}
}

// result returns the collected usage information as a usageResult.
func (uc usageCollector) result() usageResult {
	return usageResult{
		scopeRanges: uc.scopeRanges,
		usages:      uc.usages,
	}
}

// inspectBody traverses the AST of a function body to collect:
//   - Short variable declarations (x :=)
//   - Var declarations (var x int)
//   - Variable usages
//
// For each declaration, it tracks the tightest scope containing all usages,
// which determines if the declaration can be moved to a narrower scope.
func (uc usageCollector) inspectBody(body inspector.Cursor, results *ast.FieldList) {
	nodes := []ast.Node{
		// keep-sorted start
		(*ast.AssignStmt)(nil),
		(*ast.DeclStmt)(nil),
		(*ast.FuncLit)(nil),
		(*ast.Ident)(nil),
		// keep-sorted end
		(*ast.ReturnStmt)(nil),
	}

	// No need to explicitly check return statements when we have no named results
	// When the function declares result parameters, the body must end in a terminating statement,
	// so we catch any bare returns - except functions that end with a recovered panic.
	if !hasNamedResults(results) {
		nodes = nodes[:len(nodes)-1]
	}

	body.Inspect(nodes, func(c inspector.Cursor) bool {
		switch n := c.Node().(type) {
		// keep-sorted start newline_separated=yes
		case *ast.AssignStmt:
			if n.Tok != token.DEFINE {
				break // Not a short variable declaration
			}

			if kind, _ := c.ParentEdge(); kind == edge.CommClause_Comm || // Don't consider short declarations in select cases
				kind == edge.TypeSwitchStmt_Assign { // Don't consider short declarations in type switches
				break
			}

			uc.handleShortDecl(n, c.Index())

		case *ast.DeclStmt:
			gen, ok := n.Decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				break
			}

			uc.handleDeclStmt(gen, c.Index())

		case *ast.FuncLit:
			fbody, fresults := c.ChildAt(edge.FuncLit_Body, -1), n.Type.Results
			uc.inspectBody(fbody, fresults)

			return false

		case *ast.Ident:
			if n.Name == "_" {
				break
			}

			uc.handleIdent(n)

		case *ast.ReturnStmt:
			if len(n.Results) > 0 {
				break
			}

			uc.handleNamedResults(results)

			// keep-sorted end
		}

		return true
	})
}

// hasNamedResults reports whether the function has named result parameters.
func hasNamedResults(results *ast.FieldList) bool {
	return results != nil && len(results.List) > 0 && len(results.List[0].Names) > 0
}

// handleShortDecl processes short variable declarations (:=).
func (uc usageCollector) handleShortDecl(stmt *ast.AssignStmt, decl nodeIndex) {
	// The scope of a variable identifier declared inside a function begins at the end of the ShortVarDecl.
	start := stmt.End()
	// For each identifier on the LHS
	for id := range allAssigned(stmt) {
		if def, ok := uc.TypesInfo.Defs[id]; ok {
			if def == nil {
				continue // Symbolic variable in type switch (e.g., switch x := y.(type))
			}

			// Record a new variable definition
			uc.recordDeclaration(def, start, decl, id)

			continue
		}

		if use, ok := uc.TypesInfo.Uses[id]; ok {
			// Record reassignment of an existing variable
			uc.recordRedeclaration(use, start, decl, id)

			continue
		}

		uc.reportInternalError(id, "Unknown declaration for variable %s", id.Name)
	}
}

// recordDeclaration records a new variable definition from a short declaration.
// It initializes the usage tracking for this variable with its declaration position.
func (uc usageCollector) recordDeclaration(def types.Object, start token.Pos, decl nodeIndex, id *ast.Ident) {
	v, ok := def.(*types.Var)
	if !ok {
		uc.reportInternalError(id, "Non-variable declaration of %q", id.Name)
		return
	}

	// Variable declaration
	if _, ok := uc.usages[v]; ok {
		uc.reportInternalError(id, "Redeclaration of variable %q", id.Name)
	}

	usage := nodeUsage{decl: decl, used: false}
	uc.usages[v] = []nodeUsage{usage}

	uc.decls[v] = declUsage{start: start, ignore: id.NamePos}
}

// recordRedeclaration records a reassignment of an existing variable in a short declaration.
// If the variable was declared and not tracked (function parameters), it creates
// a placeholder entry with invalidNodeIndex to indicate external declaration.
func (uc usageCollector) recordRedeclaration(use types.Object, start token.Pos, decl nodeIndex, id *ast.Ident) {
	v, ok := use.(*types.Var)
	if !ok {
		return
	}

	usage := nodeUsage{decl: decl, used: false}

	if usages, ok := uc.usages[v]; ok {
		uc.usages[v] = append(usages, usage)
	} else {
		uc.usages[v] = []nodeUsage{{decl: invalidNode, used: true}, usage}
	}

	uc.decls[v] = declUsage{start: start, ignore: id.NamePos}
}

// handleDeclStmt processes var declarations (var x, y = ...).
func (uc usageCollector) handleDeclStmt(gen *ast.GenDecl, decl nodeIndex) {
	for _, spec := range gen.Specs {
		vspec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		// The scope of a variable identifier declared inside a function begins at the end of the VarSpec.
		uc.recordDeclarations(vspec, decl)
	}
}

// recordDeclarations records all variable definitions from a single ValueSpec.
func (uc usageCollector) recordDeclarations(vspec *ast.ValueSpec, decl nodeIndex) {
	start := vspec.End()
	for _, id := range vspec.Names {
		if id.Name == "_" {
			continue // blank identifier
		}

		def, ok := uc.TypesInfo.Defs[id]
		if !ok {
			uc.reportInternalError(vspec, "Non-definition of variable %s", id.Name)
			continue
		}

		uc.recordDeclaration(def, start, decl, id)
	}
}

// handleIdent processes identifier usages.
func (uc usageCollector) handleIdent(id *ast.Ident) {
	v, ok := uc.TypesInfo.Uses[id].(*types.Var)
	if !ok {
		return
	}

	decl := uc.decls[v]
	if decl.ignore == id.NamePos {
		return // ignore usage on LHS of AssignStmt
	}

	usages := uc.usages[v]
	if len(usages) == 0 {
		return
	}

	usage := &usages[len(usages)-1]

	if decl.start >= id.NamePos {
		if len(usages) < 2 {
			return
		}
		// When a variable appears on the RHS of its own reassignment (e.g., x := x + 1),
		// the usage belongs to the previous declaration, not the new one being defined.
		usage = &usages[len(usages)-2]
	}

	usage.used = true

	uc.updateUsageScope(usage.decl, v.Parent(), id)
}

// updateUsageScope updates the scope range for a variable usage.
func (uc usageCollector) updateUsageScope(decl nodeIndex, declScope *types.Scope, id *ast.Ident) {
	currentRange, hasRange := uc.scopeRanges[decl]

	if hasRange {
		if currentRange.decl != declScope {
			uc.reportInternalError(id, "Different declaration scopes recorded")
		}

		if currentRange.usage == declScope {
			return // Already at the innermost scope (can't move tighter)
		}
	}

	// Find the innermost scope containing this use
	usageScope := uc.innermost(declScope, id.NamePos)

	if hasRange {
		// Compute the minimum scope that contains all uses so far
		usageScope = uc.commonAncestor(declScope, currentRange.usage, usageScope)

		if usageScope == currentRange.usage {
			return // Unchanged
		}
	}

	// Set the target scope
	uc.scopeRanges[decl] = scopeRange{declScope, usageScope}
}

// handleNamedResults marks named result parameters as used when a bare return is encountered.
func (uc usageCollector) handleNamedResults(results *ast.FieldList) {
	for id := range allListed(results) {
		v, ok := uc.TypesInfo.Defs[id].(*types.Var)
		if !ok {
			continue
		}

		usages := uc.usages[v]
		if len(usages) == 0 {
			continue
		}

		usage := &usages[len(usages)-1]

		usage.used = true

		declScope := v.Parent()
		uc.scopeRanges[usage.decl] = scopeRange{declScope, declScope}
	}
}
