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

	"golang.org/x/tools/go/ast/edge"
	"golang.org/x/tools/go/ast/inspector"
)

// collectUnusedVariables builds a map from declaration indices to the names of
// variables that are declared but never read at that declaration site.
func collectUnusedVariables(usages map[*types.Var][]NodeUsage) map[NodeIndex][]string {
	unused := make(map[NodeIndex][]string)

	for v, nodes := range usages {
		for _, usage := range nodes {
			if usage.Flags&UsageUsed != 0 {
				continue
			}

			unused[usage.Decl] = append(unused[usage.Decl], v.Name())
		}
	}

	return unused
}

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
func usedIdentifierShadowed(info *types.Info, declCursor inspector.Cursor, declScope, safeScope *types.Scope) bool {
	declNode := declCursor.Node()
	start, end := declNode.Pos(), declNode.End()

	checked := make(map[string]struct{})

	// Find used identifiers
	for e := range declCursor.Preorder((*ast.Ident)(nil)) {
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

// checkSkippedStatements checks whether there are any statements between the node at declCursor and the target.
// It returns true if possible side effects are detected (i.e., the move might be unsafe).
//
// Note that the side effects need not be in the skipped statements, they might also be in
// the declaration itself.
func checkSkippedStatements(declCursor inspector.Cursor, targetNode ast.Node) bool {
	declNode := declCursor.Node()

	// Range to check
	start := declNode.End()
	end := targetNode.Pos()

	// Find a parent that spans start...end.
	//
	// It does not necessarily include targetNode, only everything between.
	parent := declCursor.Parent()
	for {
		if parentNode := parent.Node(); parentNode == nil || parentNode.End() >= end {
			break
		}

		parent = parent.Parent()
	}

	for s := range parent.Preorder(
		// keep-sorted start
		(*ast.AssignStmt)(nil),
		(*ast.BranchStmt)(nil),
		(*ast.CaseClause)(nil),
		(*ast.CommClause)(nil),
		(*ast.DeferStmt)(nil),
		(*ast.ExprStmt)(nil),
		(*ast.ForStmt)(nil),
		(*ast.GenDecl)(nil),
		(*ast.GoStmt)(nil),
		(*ast.IfStmt)(nil),
		(*ast.IncDecStmt)(nil),
		(*ast.LabeledStmt)(nil),
		(*ast.RangeStmt)(nil),
		(*ast.ReturnStmt)(nil),
		(*ast.SelectStmt)(nil),
		(*ast.SendStmt)(nil),
		(*ast.SwitchStmt)(nil),
		(*ast.TypeSwitchStmt)(nil),
		// keep-sorted end

		// Note regarding missing ast.Stmt types:
		// - *ast.BlockStmt is covered by its sub-statements
		// - *ast.DeclStmt is covered by *ast.GenDecl
		// - *ast.EmptyStmt has no side effects
	) {
		n := s.Node()
		if n.Pos() >= end {
			break // We've moved past the area
		}

		if n.Pos() < start {
			continue // Ignore preamble
		}

		if decl, ok := n.(*ast.GenDecl); ok {
			if decl.Tok != token.VAR {
				continue // Const and Type declarations are safe to cross
			}
		}

		return true
	}

	return false
}

// hasUsedTypeChange test whether there are a type change in the declaration
// and the changed type is used - which would change the semantics, but is seldom a problem.
// We flag it either in conservative mode, or for untyped nil, which would be a compile error.
func hasUsedTypeChange(flags UsageFlags, conservative bool) bool {
	return flags&(UsageTypeChange|UsageUsed) == UsageTypeChange|UsageUsed &&
		(conservative || flags&UsageUntypedNil != 0)
}
