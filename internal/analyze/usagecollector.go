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

	"fillmore-labs.com/scopeguard/analyzer/level"
)

// usageCollector collects variable declarations and tracks their usages within a function body.
type usageCollector struct {
	Pass          // Embedded pass for type information and error reporting
	ScopeAnalyzer // Embedded scope analyzer for scope hierarchy navigation

	// scopeRanges maps declaration indices to their scope ranges (declaration scope + usage scope).
	scopeRanges map[NodeIndex]ScopeRange

	// usages maps variables to their usages history.
	// The first entry is typically the initial declaration; subsequent entries are reassignments.
	usages map[*types.Var][]NodeUsage

	// decls maps variables to their current (re)declaration.
	decls map[*types.Var]declUsage

	// assigned maps shadowed variables.
	assigned map[*types.Var]assignInfo

	// shadowed maps shadowed variables.
	shadowed map[*types.Var]shadowInfo

	// usedAfterShadow collects usage of variables used after previously shadowed.
	usedAfterShadow ShadowUsed

	// nestedAssigned collects usage of variables assigned during assignment.
	nestedAssigned NestedAssigned

	// scopeLevel is the scope analysis level.
	scopeLevel level.Scope

	// shadowLevel is the shadow analysis level.
	shadowLevel level.Shadow

	nestedAssign level.NestedAssign
}

// declUsage tracks the scope and position of a variable's last declaration.
type declUsage struct {
	// start is the position where the variable's scope begins.
	// For short variable declarations, this is the end of the statement.
	start token.Pos

	// ignore is the position of the declaration identifier itself.
	// Usages at this position (LHS of assignment) are ignored to avoid
	// counting the declaration as a usage.
	ignore token.Pos
}

type shadowInfo struct {
	// Position shadowing starts (usually the end of the shadowing declaration) and
	// ends (invalid or the end of a new assignment)
	start, end token.Pos

	// Position of the reassigned identifier
	ignore token.Pos

	// Shadowing declaration
	decl NodeIndex
}

// assignInfo contains information about variable nested assignment.
type assignInfo struct {
	node ast.Node
	end  token.Pos
}

// shadowing determines if the provided position falls within the shadowing range defined by start and end.
func (s shadowInfo) shadowing(pos token.Pos) bool {
	return pos >= s.start && (!s.end.IsValid() || pos < s.end)
}

// newUsageCollector creates a new usage collector for analyzing a function body.
func newUsageCollector(p Pass, scopes ScopeAnalyzer, scopeLevel level.Scope, shadowLevel level.Shadow, nestedAssign level.NestedAssign) usageCollector {
	uc := usageCollector{
		Pass:          p,
		ScopeAnalyzer: scopes,
		scopeRanges:   nil,
		decls:         make(map[*types.Var]declUsage),
		usages:        make(map[*types.Var][]NodeUsage),
		assigned:      nil,
		shadowed:      nil,
		scopeLevel:    scopeLevel,
		shadowLevel:   shadowLevel,
		nestedAssign:  nestedAssign,
	}

	if scopeLevel != level.ScopeOff {
		uc.scopeRanges = make(map[NodeIndex]ScopeRange)
	}

	if shadowLevel != level.ShadowOff {
		uc.shadowed = make(map[*types.Var]shadowInfo)
	}

	if nestedAssign != level.NestedOff {
		uc.assigned = make(map[*types.Var]assignInfo)
	}

	return uc
}

// result returns the collected usage information as a usageResult.
func (uc *usageCollector) result() (UsageResult, ShadowUsed, NestedAssigned) {
	return UsageResult{
			ScopeRanges: uc.scopeRanges,
			Usages:      uc.usages,
		},
		uc.usedAfterShadow,
		uc.nestedAssigned
}

// inspectBody traverses the AST of a function body to collect:
//   - Short variable declarations (x :=)
//   - Var declarations (var x int)
//   - Variable usages
//
// For each declaration, it tracks the tightest scope containing all usages,
// which determines if the declaration can be moved to a narrower scope.
func (uc *usageCollector) inspectBody(body inspector.Cursor, funcType *ast.FuncType) {
	nodes := []ast.Node{
		// keep-sorted start
		(*ast.AssignStmt)(nil),
		(*ast.DeclStmt)(nil),
		(*ast.FuncLit)(nil),
		(*ast.Ident)(nil),
		(*ast.RangeStmt)(nil),
		// keep-sorted end
		(*ast.ReturnStmt)(nil),
	}

	results := funcType.Results

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
			switch n.Tok {
			case token.ASSIGN:
				vars := extractVars(uc.TypesInfo.Uses, n.Lhs)
				uc.handleAssignedVars(n, n.End(), vars)

			case token.DEFINE:
				if kind, _ := c.ParentEdge(); kind == edge.CommClause_Comm || // Don't consider short declarations in select cases
					kind == edge.TypeSwitchStmt_Assign { // Don't consider short declarations in type switches
					break
				}

				uc.handleShortDecl(c.Index(), n)
			}

		case *ast.DeclStmt:
			gen, ok := n.Decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				break
			}

			uc.handleDeclStmt(c.Index(), gen)

		case *ast.FuncLit:
			fbody, fresults := c.ChildAt(edge.FuncLit_Body, -1), n.Type

			// Traverse recursively with different results
			uc.inspectBody(fbody, fresults)

			return false // Visited in inspectBody, do not descend

		case *ast.Ident:
			if n.Name == "_" {
				break
			}

			uc.handleIdent(c.Index(), n)

		case *ast.RangeStmt:
			if n.Key == nil {
				break
			}

			switch n.Tok {
			case token.ASSIGN:
				vars := extractVars(uc.TypesInfo.Uses, []ast.Expr{n.Key, n.Value})
				uc.handleAssignedVars(n, n.Body.Pos(), vars)

			case token.DEFINE:
				uc.handleRangeStmt(c.Index(), n)
			}

		case *ast.ReturnStmt:
			if len(n.Results) > 0 {
				break
			}

			uc.handleNamedResults(c.Index(), results, n.Pos())

			// keep-sorted end
		}

		return true
	})
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
				flags = UsageUsed | UsageTypeChange | UsageUntypedNil

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
		uc.handleAssignedVars(stmt, assignmentDone, vars)
	}
}

// handleAssignedVars handles nested assignments and updates shadow tracking when variables are assigned.
// When a shadowed outer variable is reassigned, the shadow "ends" at that point,
// as the outer variable has a new value.
//
// Note: This is lexically based, not control-flow sensitive. An assignment inside
// an `if` block or switch `case` clears the shadow for subsequent lines.
func (uc *usageCollector) handleAssignedVars(node ast.Node, assignmentDone token.Pos, vars []assignedVar) {
	for _, vid := range vars {
		v, id := vid.Var, vid.Ident
		if assignment, ok := uc.assigned[v]; ok && node.Pos() < assignment.end {
			uc.nestedAssigned = append(uc.nestedAssigned, NestedAssign{id, assignment.node})
		} else if uc.nestedAssign != level.NestedOff {
			uc.assigned[v] = assignInfo{node: node, end: assignmentDone}
		}

		// Was the assigned variable shadowed?
		s, ok := uc.shadowed[v]
		if !ok {
			continue
		}

		switch hasEnd := s.end.IsValid(); {
		case !hasEnd:
			// set shadowing to end after this assignment
			s.ignore = id.NamePos
			s.end = assignmentDone
			uc.shadowed[v] = s

		case node.Pos() >= s.end:
			// shadowing is already done
			delete(uc.shadowed, v)

		default:
			// We already have a shadow end position, but haven't reached it in our
			// current assignment. Maybe we are in a function literal.
			// Possibly reported by nested reassignment detector.
			if assignmentDone < s.end { // Update to earlier assignment
				s.ignore = id.NamePos
				s.end = assignmentDone
				uc.shadowed[v] = s
			}
		}
	}
}

// assignedVar tracks a variable and its identifier usage.
type assignedVar struct {
	*types.Var
	*ast.Ident
}

// extractVars extracts a list of unique variable identifiers from a list of expressions.
// It overlooks anything that is not a plain variable identifier, including blank identifiers.
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

	uc.decls[v] = declUsage{start: start, ignore: id.NamePos}

	if uc.shadowLevel == level.ShadowOff {
		return
	}

	if s, ok := uc.Shadowed(v, id.NamePos).(*types.Var); ok {
		uc.shadowed[s] = shadowInfo{start: start, end: token.NoPos, decl: decl}
	}
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

	uc.decls[v] = declUsage{start: assignmentDone, ignore: id.NamePos}

	if s, ok := uc.shadowed[v]; ok {
		if !s.end.IsValid() {
			s.ignore = id.NamePos
			s.end = assignmentDone
			uc.shadowed[v] = s
		} else {
			delete(uc.shadowed, v)
		}
	}
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

	decl, ok := uc.decls[v]
	if !ok || decl.ignore == id.NamePos {
		return // ignore usage on LHS of AssignStmt
	}

	if s, ok := uc.shadowed[v]; ok && s.ignore != id.NamePos {
		if s.shadowing(id.NamePos) {
			uc.recordUsedAfterShadow(v, idx, s.decl)
		}
	}

	usage := uc.attributeDeclaration(v, decl.start < id.NamePos)
	if usage == nil {
		return
	}

	usage.Flags |= UsageUsed

	if uc.scopeLevel == level.ScopeOff {
		return
	}

	uc.updateUsageScope(usage.Decl, v.Parent(), id)
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
func (uc *usageCollector) updateUsageScope(decl NodeIndex, declScope *types.Scope, id *ast.Ident) {
	currentRange, hasRange := uc.scopeRanges[decl]

	if hasRange {
		if currentRange.Decl != declScope {
			uc.ReportInternalError(id, "Different declaration scopes recorded")
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
	uc.scopeRanges[decl] = ScopeRange{declScope, usageScope}
}

// handleNamedResults marks named result parameters as used when a bare return is encountered.
func (uc *usageCollector) handleNamedResults(idx NodeIndex, results *ast.FieldList, pos token.Pos) {
	for id := range allListed(results) {
		v, ok := uc.TypesInfo.Defs[id].(*types.Var)
		if !ok {
			continue
		}

		if s, ok := uc.shadowed[v]; ok && s.shadowing(pos) {
			uc.recordUsedAfterShadow(v, idx, s.decl)
		}

		usages := uc.usages[v]
		if len(usages) == 0 {
			continue
		}

		usage := &usages[len(usages)-1]

		usage.Flags |= UsageUsed

		if uc.scopeLevel == level.ScopeOff {
			continue
		}

		declScope := v.Parent()
		uc.scopeRanges[usage.Decl] = ScopeRange{declScope, declScope}
	}
}

// recordUsedAfterShadow tracks the usage of a variable after it has been previously shadowed.
func (uc *usageCollector) recordUsedAfterShadow(v *types.Var, use, decl NodeIndex) {
	uc.usedAfterShadow = append(uc.usedAfterShadow, ShadowUse{Var: v, Use: use, Decl: decl})

	delete(uc.shadowed, v) // record only the first usage
}
