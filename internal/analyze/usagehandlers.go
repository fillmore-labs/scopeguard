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

package analyze

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ast/inspector"
)

// handleFunc processes function parameters and results, recording their declarations.
func (uc *usageCollector) handleFunc(body inspector.Cursor, recv *ast.FieldList, typ *ast.FuncType) {
	start, decl := body.Node().Pos(), body.Parent().Index()

	for _, list := range [...]*ast.FieldList{recv, typ.Params, typ.Results} {
		if list == nil {
			continue
		}

		for _, names := range list.List {
			for _, id := range names.Names {
				if id.Name == "_" {
					continue // blank identifier
				}

				v, ok := uc.TypesInfo.Defs[id].(*types.Var)
				if !ok {
					continue
				}

				// v.Parent() == uc.TypesInfo.Scopes[typ]

				// Parameter / result declaration
				uc.current[v] = declUsage{start: start, ignore: id.NamePos}
				uc.usages[v] = []NodeUsage{{Decl: decl, Flags: UsageUsed}}

				uc.notMovable(decl, v)
			}
		}
	}
}

// handleReceiveStmt processes assignments in select communication clauses (case x := <-ch:).
func (uc *usageCollector) handleReceiveStmt(decl NodeIndex, stmt *ast.AssignStmt) {
	assignmentDone := stmt.End()

	for _, id := range stmt.Lhs {
		id, ok := id.(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}

		v, ok := uc.TypesInfo.Defs[id].(*types.Var)
		if !ok {
			uc.ReportInternalError(id, "Unknown declaration for variable %s", id.Name)
			continue
		}

		// Record a new variable definition
		uc.recordDeclaration(decl, assignmentDone, id, v)

		uc.notMovable(decl, v)
	}
}

// handleShortDecl processes short variable declarations (:=).
func (uc *usageCollector) handleShortDecl(decl NodeIndex, stmt *ast.AssignStmt) {
	// The scope of a variable identifier declared inside a function begins at the end of the ShortVarDecl.
	assignmentDone := stmt.End()

	var vars []assignedVar

	// For each identifier on the LHS
	for idx, id := range stmt.Lhs {
		id, ok := id.(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}

		if def, ok := uc.TypesInfo.Defs[id]; ok {
			if def == nil {
				continue // Symbolic variable in type switch (e.g., switch x := y.(type))
			}

			// Record a new variable definition
			uc.recordDeclaration(decl, assignmentDone, id, def)

			continue
		}

		if use, ok := uc.TypesInfo.Uses[id]; ok {
			v, ok := use.(*types.Var)
			if !ok {
				continue
			}

			flags := UsageNone

			switch assignedType := exprType(uc.TypesInfo, stmt, idx); {
			case assignedType == types.Typ[types.UntypedNil]:
				// The predeclared identifier nil cannot be used to initialize a variable with no explicit type.
				// https://go.dev/ref/spec#Variable_declarations
				flags = UsageUsedAndTypeChange | UsageUntypedNil

			case !types.Identical(v.Type(), assignedType):
				flags = UsageTypeChange
			}

			// Record reassignment of an existing variable
			uc.recordReassignment(decl, assignmentDone, id, v, flags)

			vars = appendUnique(vars, v, id)

			continue
		}

		uc.ReportInternalError(id, "Unknown declaration for variable %s", id.Name)
	}

	if len(vars) > 0 {
		uc.handleAssignedVars(decl, stmt.Pos(), assignmentDone, vars)
	}
}

// assignedVar tracks a variable and its identifier usage.
type assignedVar struct {
	*types.Var
	*ast.Ident
}

// extractVars extracts a list of unique variable identifiers from a list of expressions.
// It ignores blank identifiers and non-variable identifiers.
func extractVars(uses map[*ast.Ident]types.Object, exprs []ast.Expr) []assignedVar {
	vars := make([]assignedVar, 0, len(exprs))

	for _, expr := range exprs {
		id, ok := ast.Unparen(expr).(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}

		v, ok := uses[id].(*types.Var)
		if !ok {
			continue
		}

		// Filter out duplicate occurrences, like x, x = ...
		vars = appendUnique(vars, v, id)
	}

	return vars
}

// appendUnique appends the variable to the list if it is not already present, effectively treating the list as a set.
func appendUnique(vars []assignedVar, v *types.Var, id *ast.Ident) []assignedVar {
	for _, vid := range vars {
		if vid.Var == v {
			return vars
		}
	}

	return append(vars, assignedVar{v, id})
}

// recordDeclaration records a new variable definition from a short declaration.
// It initializes the usage tracking for this variable with its declaration position.
func (uc *usageCollector) recordDeclaration(decl NodeIndex, start token.Pos, id *ast.Ident, def types.Object) {
	v, ok := def.(*types.Var)
	if !ok {
		uc.ReportInternalError(id, "Non-variable declaration of %q", id.Name)
		return
	}

	// Variable declaration
	if _, ok := uc.usages[v]; ok {
		uc.ReportInternalError(id, "Redeclaration of variable %q", id.Name)
	}

	usage := NodeUsage{Decl: decl, Flags: UsageNone}
	uc.usages[v] = []NodeUsage{usage}

	uc.current[v] = declUsage{start: start, ignore: id.NamePos}

	uc.RecordShadowingDeclaration(uc.ScopeIndex, v, id, decl)
}

// recordReassignment records a reassignment of an existing variable.
func (uc *usageCollector) recordReassignment(decl NodeIndex, assignmentDone token.Pos, id *ast.Ident, v *types.Var, flags UsageFlags) {
	usage := NodeUsage{Decl: decl, Flags: flags}

	if usages := uc.usages[v]; len(usages) > 0 {
		uc.usages[v] = append(usages, usage)
	} else {
		// If the variable was declared and is not tracked (e.g., function parameters),
		// create a placeholder entry to indicate external declaration.
		uc.usages[v] = []NodeUsage{{Decl: InvalidNode, Flags: UsageUsed}, usage}
	}

	uc.current[v] = declUsage{start: assignmentDone, ignore: id.NamePos}

	uc.RecordAssignment(v, id, assignmentDone)
}

// handleDeclStmt processes var declarations (var x, y = ...).
func (uc *usageCollector) handleDeclStmt(decl NodeIndex, gen *ast.GenDecl) {
	for _, spec := range gen.Specs {
		vspec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		// The scope of a variable identifier declared inside a function begins at the end of the VarSpec.
		uc.recordDeclarations(decl, vspec)
	}
}

// recordDeclarations records all variable definitions from a single ValueSpec.
func (uc *usageCollector) recordDeclarations(decl NodeIndex, vspec *ast.ValueSpec) {
	start := vspec.End()
	for _, id := range vspec.Names {
		if id.Name == "_" {
			continue // blank identifier
		}

		def, ok := uc.TypesInfo.Defs[id]
		if !ok {
			uc.ReportInternalError(vspec, "Non-definition of variable %s", id.Name)
			continue
		}

		uc.recordDeclaration(decl, start, id, def)
	}
}

func (uc *usageCollector) handleRangeStmt(idx NodeIndex, stmt *ast.RangeStmt) {
	assignmentDone := stmt.Body.Lbrace
	for _, e := range []ast.Expr{stmt.Key, stmt.Value} {
		if e == nil {
			continue
		}

		id, ok := ast.Unparen(e).(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}

		if def, ok := uc.TypesInfo.Defs[id]; ok {
			// Record a new variable definition
			uc.recordDeclaration(idx, assignmentDone, id, def)

			continue
		}

		uc.ReportInternalError(id, "Unknown range declaration for variable %s", id.Name)
	}
}

// handleIdent processes identifier usages.
func (uc *usageCollector) handleIdent(idx NodeIndex, id *ast.Ident) {
	v, ok := uc.TypesInfo.Uses[id].(*types.Var)
	if !ok {
		return
	}

	decl, ok := uc.current[v]
	if !ok || decl.ignore == id.NamePos {
		return // ignore usage on LHS of AssignStmt
	}

	uc.RecordShadowedUse(v, id.NamePos, idx)

	usage := uc.attributeDeclaration(v, decl.start < id.NamePos)
	if usage == nil {
		return
	}

	usage.Flags |= UsageUsed

	uc.updateUsageScope(usage.Decl, v, id)
}

// attributeDeclaration returns the declaration that a variable usage should be attributed to.
// current indicates whether the usage occurs within the scope of the current or previous declaration.
func (uc *usageCollector) attributeDeclaration(v *types.Var, current bool) *NodeUsage {
	usages := uc.usages[v]
	switch usageCount := len(usages); {
	case current && usageCount > 0:
		// The usage is attributed to the most recent declaration.
		return &usages[usageCount-1]

	case !current && usageCount > 1:
		// When a variable appears on the RHS of its own reassignment (e.g., x := x + 1),
		// the usage belongs to the previous declaration, not the new one being defined.
		return &usages[usageCount-2] // Use penultimate declaration

	default:
		return nil // No previous declaration to attribute to
	}
}

// updateUsageScope updates the scope range for a variable usage.
func (uc *usageCollector) updateUsageScope(decl NodeIndex, v *types.Var, id *ast.Ident) {
	if uc.scopeRanges == nil {
		return
	}

	declScope := v.Parent()
	currentRange, hasRange := uc.scopeRanges[decl]

	if hasRange {
		if currentRange.Decl != declScope {
			uc.ReportInternalError(id, "Different declaration scopes recorded for '%s'", v.Name())
		}

		if currentRange.Usage == declScope {
			return // Already at the innermost scope (can't move tighter)
		}
	}

	// Find the innermost scope containing this use
	usageScope := uc.Innermost(declScope, id.NamePos)

	if hasRange {
		// Compute the minimum scope that contains all uses so far
		usageScope = uc.CommonAncestor(declScope, currentRange.Usage, usageScope)

		if usageScope == currentRange.Usage {
			return // Unchanged
		}
	}

	// Set the target scope
	uc.scopeRanges[decl] = ScopeRange{Decl: declScope, Usage: usageScope}
}

// handleNamedResults marks named result parameters as used when a bare return is encountered.
func (uc *usageCollector) handleNamedResults(idx NodeIndex, results *ast.FieldList, pos token.Pos) {
	for id := range allListed(results) {
		v, ok := uc.TypesInfo.Defs[id].(*types.Var)
		if !ok {
			continue
		}

		uc.RecordShadowedUse(v, pos, idx)

		usages := uc.usages[v]
		if len(usages) == 0 {
			continue
		}

		usage := &usages[len(usages)-1]

		usage.Flags |= UsageUsed

		uc.notMovable(usage.Decl, v)
	}
}

// notMovable marks a variable declaration as non-movable by setting its usage scope to its declaration scope.
func (uc *usageCollector) notMovable(decl NodeIndex, v *types.Var) {
	if uc.scopeRanges == nil {
		return
	}

	declScope := v.Parent()
	uc.scopeRanges[decl] = ScopeRange{declScope, declScope} // Not movable
}
