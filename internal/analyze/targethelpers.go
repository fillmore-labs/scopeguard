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
	"iter"
	"slices"

	"golang.org/x/tools/go/ast/edge"
	"golang.org/x/tools/go/ast/inspector"
)

// alreadyDeclaredInScope checks whether any identifier is already declared in the target scope.
func alreadyDeclaredInScope(safeScope *types.Scope, identifiers iter.Seq[*ast.Ident]) bool {
	for id := range identifiers {
		// Check whether the identifier already exists at that level
		if safeScope.Lookup(id.Name) != nil {
			return true
		}
	}

	return false
}

// usedIdentifierShadowed checks whether any identifier used in the declaration would be
// shadowed by a later declaration that would make the move unsafe.
func usedIdentifierShadowed(info *types.Info, c inspector.Cursor, declNode ast.Node, declScope, safeScope *types.Scope) bool {
	start, end := declNode.Pos(), declNode.End()

	checked := make(map[string]struct{})

	// Find used identifiers
	for e := range c.Preorder((*ast.Ident)(nil)) {
		// Filter out definitions and field selectors - we only care about identifier uses
		switch kind, _ := e.ParentEdge(); kind {
		case edge.AssignStmt_Lhs, // Definition or side effect
			edge.Field_Names,
			edge.SelectorExpr_Sel,
			edge.ValueSpec_Names: // Definitions
			continue
		}

		id, ok := e.Node().(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}

		if _, ok := checked[id.Name]; ok {
			continue
		}

		use, ok := info.Uses[id]
		if !ok {
			continue
		}

		if use.Pos() > start {
			// Identifier is declared within the moved statement itself, not a use from outside
			continue
		}

		// Walk up the scope chain from safeScope to declScope, looking for shadowing declarations.
		for scope := safeScope; scope != declScope; scope = scope.Parent() {
			if shadowDecl := scope.Lookup(id.Name); shadowDecl != nil && shadowDecl.Pos() < safeScope.Pos() {
				// Found a declaration in an intermediate scope that was defined before
				// the target position, which would shadow the identifier we're using
				return true
			}
		}

		// Would the identifier be shadowed by a later declaration in the same scope?
		// This handles cases like: y := x + 1; x := 2 (can't move y past the redeclaration of x)
		if shadowDecl := declScope.Lookup(id.Name); shadowDecl != nil && shadowDecl != use &&
			// Check whether the redeclaration is after our current statement (x := x is movable)
			// and before our target position
			end < shadowDecl.Pos() && shadowDecl.Pos() < safeScope.Pos() {
			// Found a later redeclaration that would shadow the identifier
			return true
		}

		checked[id.Name] = struct{}{}
	}

	return false
}

// resolveTypeIncompatibilities prevents moves that would lose type information.
//
// When a variable is reassigned with a different type inference, moving the first
// declaration would change the variable's type. This function detects such cases
// and marks them as dontFix to preserve the original type semantics.
func (t targets) resolveTypeIncompatibilities(in *inspector.Inspector, info *types.Info, usages map[*types.Var][]nodeUsage) {
	for v, nodes := range usages {
		if len(nodes) < 2 {
			continue
		}

		// Check whether the first target is a candidate
		first := nodes[0].decl
		if first == invalidNode {
			continue
		}

		target, ok := t[first]
		if !ok || target.dontFix {
			continue
		}

		// Skip if not being moved and the variable is not in the unused list.
		// In this case the declaration remains in place and no action is needed.
		if target.targetNode == nil && !slices.Contains(target.unused, v.Name()) {
			continue
		}

		next := t.findNextUsage(nodes)
		if next == invalidNode {
			continue
		}

		stmt, ok := in.At(next).Node().(*ast.AssignStmt)
		if !ok {
			continue
		}

		if typ := exprType(info, stmt, v.Name()); types.Identical(v.Type(), typ) {
			continue
		}

		// While the value is not used, the type is
		target.unused = slices.DeleteFunc(target.unused, func(name string) bool { return name == v.Name() })
		// Prevent movement
		if target.targetNode != nil {
			target.dontFix = true
		}
		t[first] = target
	}
}

// findNextUsage finds the next non-moved usage of a variable after the first declaration.
// Returns invalidNodeIndex if no such usage exists.
func (t targets) findNextUsage(usages []nodeUsage) nodeIndex {
	for i := 1; i < len(usages); i++ {
		decl := usages[i].decl

		// skip moved declarations
		if ti, ok := t[decl]; ok && !ti.dontFix {
			continue
		}

		return decl
	}

	return invalidNode
}

// exprType finds the inferred type of the assigned variable.
func exprType(info *types.Info, n *ast.AssignStmt, name string) types.Type {
	idx := assignmentIndex(n, name)
	if idx < 0 {
		return nil
	}

	switch len(n.Rhs) {
	case len(n.Lhs):
		// [types.Checker] calls `updateExprType` for untyped constants.
		return assignedType(info, n.Rhs[idx])

	case 1:
		if tuple, ok := info.Types[n.Rhs[0]].Type.(*types.Tuple); ok {
			return tuple.At(idx).Type()
		}
	}

	return nil
}

// universeRune is the object for the predeclared "rune" type.
var universeRune = types.Universe.Lookup("rune")

// assignedType returns the type of the expressions.
//
// Note that this is a simplified implementation that only handles numeric and string literals or
// identifiers denoting a constant, not all constant expressions.
func assignedType(info *types.Info, expr ast.Expr) types.Type {
	switch expr := ast.Unparen(expr).(type) {
	case *ast.BasicLit:
		switch expr.Kind {
		case token.INT:
			return types.Typ[types.Int]
		case token.FLOAT:
			return types.Typ[types.Float64]
		case token.IMAG:
			return types.Typ[types.Complex128]
		case token.CHAR:
			return universeRune.Type()
		case token.STRING:
			return types.Typ[types.String]
		}

	case *ast.Ident:
		if obj, ok := info.Uses[expr]; ok {
			return types.Default(obj.Type())
		}
	}

	return info.Types[expr].Type
}

// assignmentIndex returns the index of the variable with the given name in the LHS of an assignment.
// Returns -1 if the variable is not found.
func assignmentIndex(n *ast.AssignStmt, name string) int {
	for i, expr := range n.Lhs {
		if id, ok := expr.(*ast.Ident); ok && id.Name == name {
			return i
		}
	}

	return -1
}

// findOrphanedDeclarations identifies declarations that would become entirely unused
// after other declarations are moved. These can have all their variables replaced with '_'.
//
// This handles the case where a variable is reassigned multiple times, and moving
// the first declaration leaves subsequent assignments with no remaining reads.
func (t targets) findOrphanedDeclarations(usages map[*types.Var][]nodeUsage) map[nodeIndex][]string {
	orphanedDeclarations := make(map[nodeIndex][]string)

	for v, nodes := range usages {
		// Optimization: We need at least one moved and one non-moved
		if len(nodes) < 2 {
			continue
		}

		// Check if there are any read usages remaining
		hasUsage := false

		for _, usage := range nodes {
			index := usage.decl
			if index == invalidNode {
				hasUsage = true
				break
			}

			// skip moved declarations
			if t, ok := t[index]; ok && !t.dontFix {
				continue
			}

			if usage.used {
				hasUsage = true
				break
			}
		}

		if hasUsage {
			continue
		}

		// No usages remaining, mark all remaining occurrences for removal
		for _, usage := range nodes {
			index := usage.decl
			if index == invalidNode {
				continue
			}

			if t, ok := t[index]; ok && !t.dontFix {
				continue
			}

			orphanedDeclarations[index] = append(orphanedDeclarations[index], v.Name())
		}
	}

	return orphanedDeclarations
}
