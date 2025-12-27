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

package report

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/edge"
	"golang.org/x/tools/go/ast/inspector"

	"fillmore-labs.com/scopeguard/internal/astutil"
	"fillmore-labs.com/scopeguard/internal/target"
)

var rawcfg = &printer.Config{Mode: printer.RawFormat}

// createEdits creates a suggested fix to move a variable declaration to a tighter scope.
func createEdits(p *analysis.Pass, in *inspector.Inspector, move target.MoveTarget) []analysis.TextEdit {
	stmt := move.Decl.Node(in)

	// Get the bounds of the original statement (including comments)
	pos, end := statementBounds(stmt)

	// Handle delete-only case (unused variable removal)
	if move.TargetNode == nil {
		return removeUnused(stmt, move.Unused)
	}

	// Determine where and how to insert the declaration
	info := calcInsertInfo(p, move.TargetNode)
	if !info.pos.IsValid() {
		return nil
	}

	var (
		buf           bytes.Buffer
		extraRemovals []analysis.TextEdit
		err           error
	)

	// Build the declaration text with appropriate formatting
	if info.needsNewline {
		buf.WriteByte('\n') // ignore error
	} else {
		buf.WriteByte(' ') // ignore error
	}

	switch stmt := stmt.(type) {
	case *ast.AssignStmt:
		// Insert the statement (wrap composite literals if moving to the Init field)
		// Combine with additional declarations if present
		extraRemovals, err = fprintAssign(&buf, in, p.Fset, move, stmt, info.moveToInit)

	case *ast.DeclStmt:
		err = fprintDecl(&buf, p.Fset, stmt, move.Unused)

	default:
		err = rawcfg.Fprint(&buf, p.Fset, stmt)
	}

	if err != nil {
		astutil.InternalError(p, stmt, "Can't render statement: %s", err)

		return nil
	}

	if info.needsSemicolon {
		buf.WriteByte(';') // ignore error
	} else {
		buf.WriteByte(' ') // ignore error
	}

	// Build text edits: remove from the old location, insert at the new location
	edits := []analysis.TextEdit{
		{Pos: pos, End: end},                  // Remove from the old location
		{Pos: info.pos, NewText: buf.Bytes()}, // Insert at the target location
	}
	edits = append(edits, info.extraEdits...) // Add any additional edits (e.g., for while-style loops)
	edits = append(edits, extraRemovals...)   // Add removals for combined declarations

	return edits
}

// statementBounds returns the start and end positions of a statement, including comments.
//
// For var declarations, this includes doc comments before the declaration and line comments after it.
func statementBounds(stmt ast.Node) (pos, end token.Pos) {
	pos, end = stmt.Pos(), stmt.End()

	if declStmt, ok := stmt.(*ast.DeclStmt); ok {
		if g, ok := declStmt.Decl.(*ast.GenDecl); ok {
			// Include doc comments that appear before the var keyword
			if doc := g.Doc; doc != nil && doc.Pos() < pos {
				pos = doc.Pos()
			}

			// Include line comments that appear after the declaration
			if vspec, ok := g.Specs[len(g.Specs)-1].(*ast.ValueSpec); ok {
				if comment := vspec.Comment; comment != nil && end < comment.End() {
					end = comment.End()
				}
			}
		}
	}

	return pos, end
}

// removeUnused generates text edits to delete or replace unused variables with the blank identifier '_'.
func removeUnused(stmt ast.Node, unused []string) []analysis.TextEdit {
	switch n := stmt.(type) {
	case *ast.AssignStmt:
		return removeUnusedAssign(n, unused)

	case *ast.DeclStmt:
		return removeUnusedDecl(n, unused)
	}

	return nil
}

// removeUnusedAssign handles removal of unused variables from assignment statements (:= and =).
func removeUnusedAssign(n *ast.AssignStmt, unused []string) []analysis.TextEdit {
	if n.Tok != token.DEFINE && n.Tok != token.ASSIGN {
		return nil
	}

	var edits []analysis.TextEdit

	underscore := []byte("_")
	all := true

	for _, expr := range n.Lhs {
		id, ok := expr.(*ast.Ident)
		if !ok || id.Name == "_" {
			continue // blank identifier
		}

		if !slices.Contains(unused, id.Name) {
			all = false
			continue
		}

		edits = append(edits, analysis.TextEdit{Pos: id.Pos(), End: id.End(), NewText: underscore})
	}

	if all && n.Tok == token.DEFINE {
		// Change `:=` to `=` when all identifiers are removed
		edits = append(edits, analysis.TextEdit{Pos: n.TokPos, End: n.TokPos + 1})
	}

	return edits
}

// removeUnusedDecl handles removal of unused variables from var declaration statements.
func removeUnusedDecl(n *ast.DeclStmt, unused []string) []analysis.TextEdit {
	decl, ok := n.Decl.(*ast.GenDecl)
	if !ok || decl.Tok != token.VAR {
		return nil
	}

	var (
		edits       []analysis.TextEdit
		allSpecs    = true
		removeSpecs []*ast.ValueSpec
		underscore  = []byte("_")
	)

	for _, spec := range decl.Specs {
		vspec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		all := true

		var remove []*ast.Ident

		for _, id := range vspec.Names {
			if id.Name == "_" {
				continue // blank identifier
			}

			if !slices.Contains(unused, id.Name) {
				all = false
				continue
			}

			remove = append(remove, id)
		}

		if all && len(vspec.Values) == 0 {
			removeSpecs = append(removeSpecs, vspec)
		} else {
			allSpecs = false

			for _, id := range remove {
				edits = append(edits, analysis.TextEdit{Pos: id.Pos(), End: id.End(), NewText: underscore})
			}
		}
	}

	if allSpecs {
		edits = append(edits, analysis.TextEdit{Pos: decl.Pos(), End: decl.End()})
	} else {
		for _, vspec := range removeSpecs {
			edits = append(edits, analysis.TextEdit{Pos: vspec.Pos(), End: vspec.End()})
		}
	}

	return edits
}

// insertInfo contains all information needed to insert a declaration at a target location.
type insertInfo struct {
	pos            token.Pos           // Where to insert the declaration
	moveToInit     bool                // Whether moving to an Init field (vs. block scope)
	needsNewline   bool                // Whether to add a newline before declaration
	needsSemicolon bool                // Whether to add a semicolon after declaration
	extraEdits     []analysis.TextEdit // Additional edits (e.g., for while-style for loops)
}

const initNotEmpty = "Init is not empty"

// calcInsertInfo determines where and how to insert a declaration based on the target node type.
//
// Returns information about where and how to insert the declaration.
func calcInsertInfo(p *analysis.Pass, targetNode ast.Node) insertInfo {
	switch n := targetNode.(type) {
	case *ast.IfStmt:
		if n.Init != nil {
			astutil.InternalError(p, n.Init, initNotEmpty)
			return insertInfo{pos: token.NoPos}
		}

		return insertInfo{
			pos:            n.If + 2, // After "if"
			moveToInit:     true,
			needsSemicolon: true,
		}

	case *ast.ForStmt:
		if n.Init != nil {
			astutil.InternalError(p, n.Init, initNotEmpty)
			return insertInfo{pos: token.NoPos}
		}

		var extraEdits []analysis.TextEdit
		if n.Post == nil && n.Body != nil && n.Body.Lbrace.IsValid() {
			// While-style for loop: add semicolon before opening brace
			extraEdits = []analysis.TextEdit{{
				Pos:     n.Body.Lbrace,
				NewText: []byte("; "),
			}}
		}

		return insertInfo{
			pos:            n.For + 3, // After "for"
			moveToInit:     true,
			needsSemicolon: n.Post == nil, // While-style loop needs an extra semicolon
			extraEdits:     extraEdits,
		}

	case *ast.SwitchStmt:
		if n.Init != nil {
			astutil.InternalError(p, n.Init, initNotEmpty)
			return insertInfo{pos: token.NoPos}
		}

		return insertInfo{
			pos:            n.Switch + 6, // After "switch"
			moveToInit:     true,
			needsSemicolon: true,
		}

	case *ast.TypeSwitchStmt:
		if n.Init != nil {
			astutil.InternalError(p, n.Init, initNotEmpty)
			return insertInfo{pos: token.NoPos}
		}

		return insertInfo{
			pos:            n.Switch + 6, // After "switch"
			moveToInit:     true,
			needsSemicolon: true,
		}

	case *ast.BlockStmt:
		return insertInfo{
			pos:          n.Lbrace + 1, // After the opening brace
			needsNewline: true,
		}

	case *ast.CaseClause:
		return insertInfo{
			pos:          n.Colon + 1, // After the ':'
			needsNewline: true,
		}

	case *ast.CommClause:
		return insertInfo{
			pos:          n.Colon + 1, // After the ':'
			needsNewline: true,
		}

	default:
		astutil.InternalError(p, n, "Invalid target")
		return insertInfo{pos: token.NoPos}
	}
}

// fprintAssign prints an assignment statement.
func fprintAssign(buf *bytes.Buffer, in *inspector.Inspector, fset *token.FileSet, move target.MoveTarget, stmt *ast.AssignStmt, moveToInit bool) ([]analysis.TextEdit, error) {
	// If we are not moving to Init (which might require wrapping composite literals) AND we have no other decls to combine,
	// we can use the statement as is.
	if stmt.Tok != token.DEFINE || (!moveToInit && len(move.Unused) == 0 && len(move.AbsorbedDecls) == 0) {
		return nil, rawcfg.Fprint(buf, fset, stmt)
	}

	// We handle composite literal wrapping for the RHS if moving to Init
	var cls []int
	if moveToInit {
		cls = compositeLits(cls, move.Decl.Cursor(in), 0)
	}

	// Start with the initial statement's LHS and RHS
	var lhs []ast.Expr

	for _, expr := range stmt.Lhs {
		if id, ok := expr.(*ast.Ident); ok && slices.Contains(move.Unused, id.Name) {
			expr = &ast.Ident{NamePos: id.NamePos, Name: "_"}
		}

		lhs = append(lhs, expr)
	}

	rhs := slices.Clone(stmt.Rhs)

	var extraRemovals []analysis.TextEdit
	// Combine components from additional declarations
	for _, otherDecl := range move.AbsorbedDecls {
		otherCursor := otherDecl.Decl.Cursor(in)
		otherNode := otherCursor.Node()

		otherAssign, ok := otherNode.(*ast.AssignStmt)
		if !ok {
			return nil, fmt.Errorf("unexpected node type: %T", otherNode) // Should not happen
		}

		// Add removal edit for this declaration
		pos, end := statementBounds(otherNode)
		extraRemovals = append(extraRemovals, analysis.TextEdit{Pos: pos, End: end})

		if moveToInit {
			cls = compositeLits(cls, otherCursor, len(rhs))
		}

		// Append LHS and RHS
		for _, expr := range otherAssign.Lhs {
			if id, ok := expr.(*ast.Ident); ok && slices.Contains(otherDecl.Unused, id.Name) {
				expr = &ast.Ident{NamePos: id.NamePos, Name: "_"}
			}

			lhs = append(lhs, expr)
		}

		rhs = append(rhs, otherAssign.Rhs...)
	}

	// Manual printing of assignment to avoid spurious newlines and handle formatting
	if err := fprintAssignLHS(buf, fset, lhs, move.Unused); err != nil {
		return nil, err
	}

	buf.WriteByte(' ')                 // ignore error
	buf.WriteString(stmt.Tok.String()) // ignore error
	buf.WriteByte(' ')                 // ignore error

	if err := fprintAssignRHS(buf, fset, rhs, cls); err != nil {
		return nil, err
	}

	return extraRemovals, nil
}

// fprintAssignLHS prints the left-hand side of an assignment, replacing unused variables with '_'.
func fprintAssignLHS(buf *bytes.Buffer, fset *token.FileSet, lhs []ast.Expr, unused []string) error {
	for i, expr := range lhs {
		if i > 0 {
			buf.WriteString(", ") // ignore error
		}

		// Replace unused variables with '_'
		if id, ok := expr.(*ast.Ident); ok && slices.Contains(unused, id.Name) {
			expr = &ast.Ident{NamePos: id.NamePos, Name: "_"}
		}

		if err := rawcfg.Fprint(buf, fset, expr); err != nil {
			return err
		}
	}

	return nil
}

// fprintAssignRHS prints the right-hand side of an assignment.
func fprintAssignRHS(buf *bytes.Buffer, fset *token.FileSet, rhs []ast.Expr, cls []int) error {
	for i, expr := range rhs {
		if i > 0 {
			buf.WriteString(", ") // ignore error
		}

		if slices.Contains(cls, i) {
			expr = &ast.ParenExpr{Lparen: expr.Pos(), X: expr, Rparen: expr.End()}
		}

		if err := rawcfg.Fprint(buf, fset, expr); err != nil {
			return err
		}
	}

	return nil
}

// fprintDecl prints a declaration statement, filtering out unused value specs.
func fprintDecl(buf *bytes.Buffer, fset *token.FileSet, stmt *ast.DeclStmt, unused []string) error {
	if len(unused) == 0 {
		return rawcfg.Fprint(buf, fset, stmt)
	}

	decl, ok := stmt.Decl.(*ast.GenDecl)
	if !ok || decl.Tok != token.VAR {
		return rawcfg.Fprint(buf, fset, stmt)
	}

	specs := make([]ast.Spec, 0, len(decl.Specs))
	for _, spec := range decl.Specs {
		vspec, ok := spec.(*ast.ValueSpec)
		if !ok {
			specs = append(specs, spec)

			continue
		}

		hasValues := len(vspec.Values) > 0

		names := make([]*ast.Ident, 0, len(vspec.Names))
		for _, id := range vspec.Names {
			if slices.Contains(unused, id.Name) {
				if !hasValues {
					continue
				}

				id = &ast.Ident{NamePos: id.NamePos, Name: "_"}
			}

			names = append(names, id)
		}

		if len(names) > 0 {
			specs = append(specs,
				&ast.ValueSpec{
					Doc:     vspec.Doc,
					Names:   names,
					Type:    vspec.Type,
					Values:  vspec.Values,
					Comment: vspec.Comment,
				})
		}
	}

	if len(specs) == 0 {
		return nil
	}

	stmt = &ast.DeclStmt{
		Decl: &ast.GenDecl{
			Doc:    decl.Doc,
			TokPos: decl.TokPos,
			Tok:    decl.Tok,
			Lparen: decl.Lparen,
			Specs:  specs,
			Rparen: decl.Rparen,
		},
	}

	return rawcfg.Fprint(buf, fset, stmt)
}

// compositeLits identifies which RHS expressions in an assignment contain [composite literals] that need parenthesization:
//
//	A parsing ambiguity arises when a composite literal [...] appears as an operand between the keyword and the opening brace of the block of an "if", "for", or "switch" statement, ...
//
// [composite literals]: https://go.dev/ref/spec#Composite_literals
func compositeLits(cls []int, c inspector.Cursor, start int) []int {
	index := start
	// Iterate through each RHS expression by index
	for e, hasNode := c.ChildAt(edge.AssignStmt_Rhs, 0), true; hasNode; e, hasNode = e.NextSibling() {
		if NeedParent(e) {
			// Record the index of this RHS expression
			cls = append(cls, index)
		}

		index++
	}

	return cls
}

// NeedParent detects whether an expression contains composite literals that need parenthesization.
func NeedParent(e inspector.Cursor) bool {
	// If the expression root itself is a composite literal, it has no enclosing parents
	// within the expression boundary to provide safe delimiters. It needs parenthesization.
	if _, ok := e.Node().(*ast.CompositeLit); ok {
		return true
	}

compLits:
	for c := range e.Preorder((*ast.CompositeLit)(nil)) {
		// Found a composite literal. Walk up the parent chain to check if it's already
		// safely delimited by parentheses, block braces, or other constructs.
		for p := c; p.Index() != e.Index(); p = p.Parent() {
			switch kind, _ := p.ParentEdge(); kind {
			// Already wrapped
			case edge.ParenExpr_X,
				// Inside a block statement, function call or index expression
				edge.BlockStmt_List, edge.CallExpr_Args, edge.IndexExpr_Index,
				// Slice expression
				edge.SliceExpr_Low, edge.SliceExpr_High, edge.SliceExpr_Max,
				// Nested composite literal
				edge.CompositeLit_Elts, edge.KeyValueExpr_Value:
				// Safely delimited, check next composite literal
				continue compLits
			}
		}

		// Reached the root expression without finding delimiters
		return true
	}

	// No problematic composite literals found
	return false
}
