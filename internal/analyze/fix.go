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
//
// This function orchestrates the fix generation by:
//  1. Determining insertion location via getInsertInfo
//  2. Preparing the statement (wrapping composite literals if needed)
//  3. Building the text edits to perform the move
//
// Parameters:
//   - c: inspector.Cursor pointing to the declaration statement to move
//   - targetNode: The AST node representing the target scope (IfStmt, ForStmt, BlockStmt, etc.)
//   - message: The diagnostic message to include in the suggested fix
//
// Returns:
//   - analysis.SuggestedFix: The fix with text edits
//   - bool: true if a valid fix was generated, false otherwise
func (p pass) createEdits(
	c inspector.Cursor,
	targetNode ast.Node,
) []analysis.TextEdit {
	stmt := c.Node()

	// Get the bounds of the original statement (including comments)
	pos, end := p.getStatementBounds(stmt)

	// Handle delete-only case (unused variable removal)
	if targetNode == nil {
		return []analysis.TextEdit{{Pos: pos, End: end, NewText: []byte{}}}
	}

	// Determine where and how to insert the declaration
	info, ok := p.calcInsertInfo(targetNode, stmt)
	if !ok {
		return nil
	}

	// Prepare the statement for insertion (wrap composite literals if moving to Init field)
	stmtToInsert := p.prepareStatement(c, stmt, info.moveToInit)

	// Build the declaration text with appropriate formatting
	var buf bytes.Buffer
	if info.needsNewline {
		buf.WriteByte('\n')
	} else {
		buf.WriteByte(' ')
	}

	if err := rawcfg.Fprint(&buf, p.pass.Fset, stmtToInsert); err != nil {
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
// This function encapsulates all the logic for determining insertion position and formatting
// requirements for different target scope types.
//
// Parameters:
//   - targetNode: The AST node representing the target scope (IfStmt, ForStmt, BlockStmt, etc.)
//   - stmt: The declaration statement being moved (for validation)
//
// Returns:
//   - insertInfo: Information about where and how to insert the declaration
//   - bool: true if a valid insert location was determined, false otherwise
func (p pass) calcInsertInfo(targetNode, stmt ast.Node) (insertInfo, bool) {
	switch n := targetNode.(type) {
	case *ast.IfStmt:
		if !p.validTarget(n.Init, stmt) {
			return insertInfo{}, false
		}

		return insertInfo{
			pos:            n.If + 2, // After "if"
			moveToInit:     true,
			needsSemicolon: true,
		}, true

	case *ast.ForStmt:
		if !p.validTarget(n.Init, stmt) {
			return insertInfo{}, false
		}

		info := insertInfo{
			pos:            n.For + 3, // After "for"
			moveToInit:     true,
			needsSemicolon: n.Post == nil, // While-style loop needs extra semicolon
		}
		// While-style for loop: add semicolon before opening brace
		if n.Post == nil && n.Body != nil && n.Body.Lbrace.IsValid() {
			info.extraEdits = []analysis.TextEdit{{
				Pos:     n.Body.Lbrace,
				NewText: []byte("; "),
			}}
		}

		return info, true

	case *ast.SwitchStmt:
		if !p.validTarget(n.Init, stmt) {
			return insertInfo{}, false
		}

		return insertInfo{
			pos:            n.Switch + 6, // After "switch"
			moveToInit:     true,
			needsSemicolon: true,
		}, true

	case *ast.TypeSwitchStmt:
		if !p.validTarget(n.Init, stmt) {
			return insertInfo{}, false
		}

		return insertInfo{
			pos:            n.Switch + 6, // After "switch"
			moveToInit:     true,
			needsSemicolon: true,
		}, true

	case *ast.BlockStmt:
		return insertInfo{
			pos:          n.Lbrace + 1, // After the opening brace
			needsNewline: true,
		}, true

	case *ast.CaseClause:
		return insertInfo{
			pos:          n.Colon + 1, // After the ':'
			needsNewline: true,
		}, true

	case *ast.CommClause:
		return insertInfo{
			pos:          n.Colon + 1, // After the ':'
			needsNewline: true,
		}, true

	default:
		return insertInfo{}, false
	}
}

// getStatementBounds returns the start and end positions of a statement, including comments.
//
// For var declarations, this includes doc comments before the declaration and line comments after it.
func (p pass) getStatementBounds(stmt ast.Node) (pos, end token.Pos) {
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
//
// When moving to Init fields, composite literals must be wrapped in parentheses
// to avoid parsing ambiguity with block braces.
func (p pass) prepareStatement(c inspector.Cursor, stmt ast.Node, moveToInit bool) ast.Node {
	// Only need to prepare assignment statements when moving to Init fields
	assignStmt, ok := stmt.(*ast.AssignStmt)
	if !ok || !moveToInit {
		return stmt
	}

	// Check which RHS expressions are composite literals
	cls := compositeLits(c)
	if len(cls) == 0 {
		return stmt
	}

	// Wrap composite literals in parentheses
	rhs := make([]ast.Expr, 0, len(assignStmt.Rhs))
	for i, expr := range assignStmt.Rhs {
		if slices.Contains(cls, i) {
			expr = &ast.ParenExpr{X: expr}
		}

		rhs = append(rhs, expr)
	}

	return &ast.AssignStmt{
		Lhs:    assignStmt.Lhs,
		TokPos: assignStmt.TokPos,
		Tok:    assignStmt.Tok,
		Rhs:    rhs,
	}
}

// compositeLits identifies which RHS expressions in an assignment are [composite literals] that need parenthesization.
//
// When a composite literal appears in an Init field (if/for/switch), the opening brace can be mistaken
// for the block's opening brace, causing a parse error. Go requires parentheses to disambiguate.
//
// This function detects which RHS expressions are composite literals, so generateFix can wrap them
// in parentheses when moving declarations to Init fields.
//
// Parameters:
//   - c: inspector.Cursor pointing to an *ast.AssignStmt
//
// Returns:
//   - []int: Indices of RHS expressions that are composite literals requiring parentheses
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

// validTarget checks if a target Init field is occupied and reports an internal error if so.
//
// This function is a safety check that should never trigger in normal operation. It's called when
// generateFix attempts to move a declaration to an Init field. If the Init field is already occupied,
// it means either:
//  1. The analyzer's logic has a bug (it should have detected the occupied Init in canUseNode)
//  2. The code was modified between analysis and fix generation
//
// Parameters:
//   - target: The Init field AST node that we want to use (e.g., IfStmt.Init)
//   - source: The declaration statement we're trying to move
//
// Returns:
//   - bool: true if init is free (valid), false if init is occupied (invalid)
func (p pass) validTarget(init, source analysis.Range) bool {
	if init == nil {
		return true
	}

	// Target Init field is already occupied - this should never happen
	message := "Internal error: Init is not empty"
	related := []analysis.RelatedInformation{
		{Pos: source.Pos(), End: source.End(), Message: "Move candidate"},
	}

	p.pass.Report(analysis.Diagnostic{
		Pos:     init.Pos(),
		End:     init.End(),
		Message: message,
		Related: related,
	})

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
