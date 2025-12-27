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

// isScopeMoveValid checks if the safe scope is valid for moving.
func isScopeMoveValid(p Pass, safeScope, declScope *types.Scope, declNode ast.Node) bool {
	switch safeScope {
	case declScope:
		return false // No scope tightening possible

	case nil:
		p.ReportInternalError(declNode, "Invalid scope calculations")
		return false

	default:
		return true
	}
}

// declInfo extracts assigned identifiers and whether the move is restricted to block statements only.
func declInfo(declNode ast.Node, cf CurrentFile, maxLines int) (identifiers iter.Seq[*ast.Ident], onlyBlock bool) {
	switch n := declNode.(type) {
	case *ast.AssignStmt:
		// Short declarations can go to init fields if they're small enough
		return allAssigned(n), maxLines > 0 && cf.Lines(declNode) > maxLines

	case *ast.DeclStmt:
		// var declarations can only go to block statements (not init fields)
		return allDeclared(n), true

	default:
		// Unsupported declaration type
		return nil, false
	}
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

	// Track which identifiers we've already checked to avoid redundant work
	checked := make(map[string]struct{})

	// Traverse all identifiers used in the declaration
	for e := range declCursor.Preorder((*ast.Ident)(nil)) {
		// Filter out definitions and field selectors - we only care about identifier uses
		switch kind, _ := e.ParentEdge(); kind {
		case edge.AssignStmt_Lhs, // Left-hand side of assignment (definition)
			edge.Field_Names,      // Struct field names
			edge.SelectorExpr_Sel, // Right side of dot selector (x.Field)
			edge.ValueSpec_Names:  // Variable declaration names
			continue
		}

		id, ok := e.Node().(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}

		// Skip if we've already checked this identifier
		if _, ok := checked[id.Name]; ok {
			continue
		}

		// Get the object this identifier refers to
		use, ok := info.Uses[id]
		if !ok {
			continue
		}

		// Skip identifiers declared within the statement itself
		// (e.g., in "x, y := f()", x and y are declared here, not uses)
		if use.Pos() > start {
			continue
		}

		// Intermediate scope shadowing
		// Walk up the scope chain from safeScope to declScope, looking for shadowing declarations.
		for scope := safeScope; scope != declScope; scope = scope.Parent() {
			if shadowDecl := scope.Lookup(id.Name); shadowDecl != nil && shadowDecl.Pos() < safeScope.Pos() {
				// Found a declaration in an intermediate scope that was defined before
				// the target position, which would shadow the identifier we're using
				return true
			}
		}

		// Same-scope redeclaration shadowing.
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

// enclosingInterval determines the execution interval [start, end) between the declaration
// and the target node, and finds the innermost parent that spans this interval.
func enclosingInterval(declCursor inspector.Cursor, targetNode ast.Node) (parent inspector.Cursor, start, end token.Pos) {
	start, end = declCursor.Node().End(), targetNode.Pos()

	parent = declCursor.Parent()
	for !extendsUntil(parent, end) {
		parent = parent.Parent()
	}

	return parent, start, end
}

// extendsUntil returns true when the current parent node extends until end.
func extendsUntil(parent inspector.Cursor, end token.Pos) bool {
	parentNode := parent.Node()

	return parentNode == nil || parentNode.End() >= end
}

// SideEffectsInInterval checks whether there are any statements between two nodes
// that might have side effects. It returns true if such statements are detected.
//
// The check covers the interval [start, end), excluding the end position.
func SideEffectsInInterval(info *types.Info, parent inspector.Cursor, start, end token.Pos) bool {
	// Iterate over all nodes in the parent to find statements in the interval.
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
			break // We've moved past the area of interest
		}

		if n.End() <= start {
			continue // Before the start of the interval
		}

		if decl, ok := n.(*ast.GenDecl); ok && !varSpecWithValue(info, decl) {
			continue // Safe declaration
		}

		return true // Found a statement with potential side effects
	}

	return false
}

// varSpecWithValue checks if a GenDecl AST node represents a `var` declaration that includes initialization values.
func varSpecWithValue(info *types.Info, decl *ast.GenDecl) bool {
	if decl.Tok != token.VAR { // type declaration and const are safe
		return false
	}

	for _, spec := range decl.Specs {
		// A ValueSpec with values implies execution (initialization).
		if spec, ok := spec.(*ast.ValueSpec); ok {
			for _, val := range spec.Values {
				// Check for new(...)
				if c, ok := ast.Unparen(val).(*ast.CallExpr); ok {
					if !builtinNew(info, c) {
						return true
					}

					continue
				}

				// Check for constants
				if tv, ok := info.Types[val]; ok && tv.Value != nil {
					continue
				}

				return true
			}
		}
	}

	return false
}

// builtinNew checks if the call expression is a call to the built-in `new` function.
func builtinNew(info *types.Info, c *ast.CallExpr) bool {
	id, ok := c.Fun.(*ast.Ident)
	if !ok || id.Name != "new" {
		return false
	}

	if _, ok := info.Uses[id].(*types.Builtin); !ok {
		return false
	}

	return true
}

// initField determines whether the targetNode is an initialization field in a control structure.
func initField(targetNode ast.Node) bool {
	switch targetNode.(type) {
	case *ast.IfStmt,
		*ast.ForStmt,
		*ast.SwitchStmt,
		*ast.TypeSwitchStmt:
		return true

	default:
		return false
	}
}

// usedAndTypeChange tests whether a type change in a declaration would affect semantics.
func usedAndTypeChange(flags UsageFlags, conservative bool) bool {
	// Check if both Used and TypeChange flags are set
	usedAndTypeChange := flags&UsageUsedAndTypeChange == UsageUsedAndTypeChange

	// Block in conservative mode or when untyped nil is involved
	return usedAndTypeChange && (conservative || flags&UsageUntypedNil != 0)
}
