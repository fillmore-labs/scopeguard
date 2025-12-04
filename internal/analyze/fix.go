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
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/edge"
	"golang.org/x/tools/go/ast/inspector"
)

var rawcfg = &printer.Config{Mode: printer.RawFormat}

// createEdits creates a suggested fix to move a variable declaration to a tighter scope.
func (p pass) createEdits(c inspector.Cursor, targetNode ast.Node, unused []string) []analysis.TextEdit {
	stmt := c.Node()

	// Get the bounds of the original statement (including comments)
	pos, end := statementBounds(stmt)

	// Handle delete-only case (unused variable removal)
	if targetNode == nil {
		return removeUnused(stmt, unused)
	}

	// Determine where and how to insert the declaration
	info := p.calcInsertInfo(targetNode, stmt)
	if !info.pos.IsValid() {
		return nil
	}

	// Prepare the statement for insertion (wrap composite literals if moving to Init field)
	stmtToInsert := prepareStatement(c, stmt, info.moveToInit, unused)

	// Build the declaration text with appropriate formatting
	var buf bytes.Buffer
	if info.needsNewline {
		buf.WriteByte('\n')
	} else {
		buf.WriteByte(' ')
	}

	if err := rawcfg.Fprint(&buf, p.Fset, stmtToInsert); err != nil {
		p.reportInternalError(stmt, "Can't render statement: %s", err)
		return nil
	}

	if info.needsSemicolon {
		buf.WriteByte(';')
	} else {
		buf.WriteByte(' ')
	}

	// Build text edits: remove from old location, insert at new location
	edits := []analysis.TextEdit{
		{Pos: pos, End: end, NewText: []byte{}}, // Remove from old location
		{Pos: info.pos, NewText: buf.Bytes()},   // Insert at target location
	}
	edits = append(edits, info.extraEdits...) // Add any additional edits (e.g., for while-style loops)

	return edits
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

	for id := range allAssigned(n) {
		if !slices.Contains(unused, id.Name) {
			all = false
			continue
		}

		edits = append(edits, analysis.TextEdit{Pos: id.Pos(), End: id.End(), NewText: underscore})
	}

	if all && n.Tok == token.DEFINE {
		// Change := to = when all identifiers are removed
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
	moveToInit     bool                // Whether moving to an Init field (vs block scope)
	needsNewline   bool                // Whether to add newline before declaration
	needsSemicolon bool                // Whether to add semicolon after declaration
	extraEdits     []analysis.TextEdit // Additional edits (e.g., for while-style for loops)
}

// calcInsertInfo determines where and how to insert a declaration based on the target node type.
//
// Returns information about where and how to insert the declaration.
func (p pass) calcInsertInfo(targetNode, stmt ast.Node) insertInfo {
	switch n := targetNode.(type) {
	case *ast.IfStmt:
		if n.Init != nil {
			p.reportInvalidTarget(n.Init, stmt)
			return insertInfo{pos: token.NoPos}
		}

		return insertInfo{
			pos:            n.If + 2, // After "if"
			moveToInit:     true,
			needsSemicolon: true,
		}

	case *ast.ForStmt:
		if n.Init != nil {
			p.reportInvalidTarget(n.Init, stmt)
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
			needsSemicolon: n.Post == nil, // While-style loop needs extra semicolon
			extraEdits:     extraEdits,
		}

	case *ast.SwitchStmt:
		if n.Init != nil {
			p.reportInvalidTarget(n.Init, stmt)
			return insertInfo{pos: token.NoPos}
		}

		return insertInfo{
			pos:            n.Switch + 6, // After "switch"
			moveToInit:     true,
			needsSemicolon: true,
		}

	case *ast.TypeSwitchStmt:
		if n.Init != nil {
			p.reportInvalidTarget(n.Init, stmt)
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
		return insertInfo{pos: token.NoPos}
	}
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

// prepareStatement prepares a statement for insertion at a target location.
func prepareStatement(c inspector.Cursor, stmt ast.Node, moveToInit bool, unused []string) ast.Node {
	switch stmt := stmt.(type) {
	case *ast.AssignStmt:
		return prepareAssign(c, stmt, moveToInit, unused)

	case *ast.DeclStmt:
		return prepareDecl(stmt, unused)

	default:
		return stmt
	}
}

// prepareAssign prepares an assignment statement for insertion, handling composite literal wrapping
// and unused variable replacement.
func prepareAssign(c inspector.Cursor, stmt *ast.AssignStmt, moveToInit bool, unused []string) ast.Node {
	if stmt.Tok != token.DEFINE && stmt.Tok != token.ASSIGN {
		return stmt
	}

	var cls []int
	if moveToInit {
		// Check which RHS expressions are composite literals
		cls = compositeLits(c)
	}

	if len(unused) == 0 && len(cls) == 0 {
		return stmt
	}

	var lhs []ast.Expr
	if len(unused) > 0 {
		lhs = make([]ast.Expr, len(stmt.Lhs))
		for i, expr := range stmt.Lhs {
			if id, ok := expr.(*ast.Ident); ok && slices.Contains(unused, id.Name) {
				expr = &ast.Ident{NamePos: id.NamePos, Name: "_"}
			}

			lhs[i] = expr
		}
	} else {
		lhs = stmt.Lhs
	}

	// When moving to Init fields, composite literals must be wrapped in parentheses
	// to avoid parsing ambiguity with block braces.
	var rhs []ast.Expr
	if len(cls) > 0 {
		// Wrap composite literals in parentheses
		rhs = make([]ast.Expr, len(stmt.Rhs))
		for i, expr := range stmt.Rhs {
			if slices.Contains(cls, i) {
				expr = &ast.ParenExpr{X: expr}
			}

			rhs[i] = expr
		}
	} else {
		rhs = stmt.Rhs
	}

	return &ast.AssignStmt{
		Lhs:    lhs,
		TokPos: stmt.TokPos,
		Tok:    stmt.Tok,
		Rhs:    rhs,
	}
}

// prepareDecl prepares a declaration statement for insertion, filtering out unused value specs.
func prepareDecl(stmt *ast.DeclStmt, unused []string) ast.Node {
	if len(unused) == 0 {
		return stmt
	}

	decl, ok := stmt.Decl.(*ast.GenDecl)
	if !ok || decl.Tok != token.VAR {
		return stmt
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
		return &ast.EmptyStmt{Implicit: true}
	}

	return &ast.DeclStmt{
		Decl: &ast.GenDecl{
			Doc:    decl.Doc,
			TokPos: decl.TokPos,
			Tok:    decl.Tok,
			Lparen: decl.Lparen,
			Specs:  specs,
			Rparen: decl.Rparen,
		},
	}
}

// compositeLits identifies which RHS expressions in an assignment are [composite literals] that need parenthesization.
//
// # Returns indices of RHS expressions that are composite literals requiring parentheses.
//
// [composite literals]: https://go.dev/ref/spec#Composite_literals
func compositeLits(c inspector.Cursor) []int {
	var (
		cls     []int
		index   = 0
		hasNode = true
	)

	// Iterate through each RHS expression by index
	for e := c.ChildAt(edge.AssignStmt_Rhs, 0); hasNode; e, hasNode = e.NextSibling() {
		if hasCompositeLit(e) {
			// Record the index of this RHS expression
			cls = append(cls, index)
		}

		index++
	}

	return cls
}

// hasCompositeLit detects whether an expression contains composite literals that need parenthesization.
//
// Returns true if the expression contains any composite literal that isn't already safely delimited.
func hasCompositeLit(e inspector.Cursor) (compositeLit bool) {
expressions:
	for c := range e.Preorder((*ast.CompositeLit)(nil)) {
		// Found a composite literal
		if c == e {
			// If the expression root itself is a composite literal, it has no enclosing parents
			// within the expression boundary to provide safe delimiters. It needs parenthesization.
			return true
		}

		// Walk up the parent chain to check if it's already safely delimited by parentheses,
		// block braces, or other delimiting constructs.
		for p := c.Parent(); p != e; p = p.Parent() {
			switch n := p.Node().(type) {
			case *ast.ParenExpr, // Already wrapped in parentheses: (T{1})
				*ast.BlockStmt: // Inside a block statement (e.g., function literal body): func() { return T{1} }
				continue expressions

			case *ast.FieldList: // Inside a field list (struct fields, function params) with delimiters
				if n.Opening.IsValid() { // Field list has opening delimiter (paren or brace): struct{ T{1} } or f(T{1})
					continue expressions
				}
			}
		}

		// Found a composite literal without safe delimiters - needs parenthesization
		return true
	}

	// No problematic composite literals found
	return false
}

// scopeTypeName returns a human-readable name for the scope type.
func scopeTypeName(node ast.Node) string {
	switch node.(type) {
	// keep-sorted start newline_separated=yes
	case *ast.BlockStmt:
		return "block"

	case *ast.CaseClause:
		return "case"

	case *ast.CommClause:
		return "select case"

	case *ast.File:
		return "file"

	case *ast.ForStmt:
		return "for"

	case *ast.FuncType:
		return "function"

	case *ast.IfStmt:
		return "if"

	case *ast.RangeStmt:
		return "range"

	case *ast.SwitchStmt:
		return "switch"

	case *ast.TypeSwitchStmt:
		return "type switch"

	case nil:
		return "<nil>"

	default:
		return fmt.Sprintf("nested: %T", node)
		// keep-sorted end
	}
}
