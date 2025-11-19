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
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"runtime/trace"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
)

// report generates and emits diagnostics for variables that can be moved to tighter scopes.
//
// This is the final phase of the analyzer pipeline. For each move target identified by the
// target phase, this function:
//  1. Constructs a diagnostic message describing what can be moved and where
//  2. Generates a suggested fix with text edits to perform the move (if possible)
//  3. Reports the diagnostic to the analysis framework
//
// Parameters:
//   - ctx: Context for tracing and cancellation
//   - targets: Sorted list of declarations that can be moved to tighter scopes
//   - includeGenerated: When false, identifies generated files and disables fixes for them.
//     When true, assumes generated files were already excluded by declarations phase.
//
// Returns:
//   - error: Non-nil if the inspector result is missing (should never happen in practice)
//
// Diagnostic Format:
//   - Message: "Variable(s) 'name' can be moved to tighter <scope> scope"
//   - Position: The declaration statement being flagged
//   - Related Info: Points to the target scope with message "To this <scope> scope"
//   - Suggested Fix: Text edits to move the declaration (omitted if dontFix flag is set or file is generated)
func (p pass) report(ctx context.Context, in *inspector.Inspector, targets targetResult, includeGenerated bool) error {
	defer trace.StartRegion(ctx, "report").End()

	var checker generatedChecker
	if includeGenerated { // When generated files are not included, we don't need to check
		checker = make(generatedChecker)
	}

	for _, move := range targets.move {
		p.reportAndFix(in, checker, move)
	}

	return nil
}

func (p pass) reportAndFix(in *inspector.Inspector, checker generatedChecker, move moveTarget) {
	c := in.At(move.decl)
	stmt := c.Node()

	dontFix := move.dontFix

	if f := enclosingFile(c); f != nil {
		if hasNoLintComment(p.pass.Fset, f, stmt) {
			return
		}

		if !dontFix && checker.isGenerated(f) {
			dontFix = true
		}
	}

	varNames := collectNames(stmt)

	diagnostic := analysis.Diagnostic{
		Pos: stmt.Pos(),
		End: stmt.End(),
	}

	diagnostic.Message, diagnostic.Related = createMessage(varNames, move.targetNode)

	if !dontFix {
		if edits := p.createEdits(c, move.targetNode); len(edits) > 0 {
			diagnostic.SuggestedFixes = []analysis.SuggestedFix{{Message: diagnostic.Message, TextEdits: edits}}
		}
	}

	p.pass.Report(diagnostic)
}

func hasNoLintComment(fset *token.FileSet, f *ast.File, stmt ast.Node) bool {
	for _, co := range f.Comments {
		if co.Pos() > stmt.Pos() {
			if fset.Position(co.Pos()).Line != fset.Position(stmt.Pos()).Line {
				break
			}

			if linters, ok := parseDirective(co.List[0].Text); !ok || !containsScopeguard(linters) {
				break
			}

			return true
		}
	}

	return false
}

func enclosingFile(c inspector.Cursor) *ast.File {
	for e := range c.Enclosing((*ast.File)(nil)) {
		f, _ := e.Node().(*ast.File)

		return f
	}

	return nil
}

var nolintPattern = regexp.MustCompile(`^//\s*nolint:([a-zA-Z0-9,_-]+)`)

// parseDirective extracts linter names from a nolint comment.
func parseDirective(text string) (linters []string, ok bool) {
	matches := nolintPattern.FindStringSubmatch(text)
	if matches == nil {
		return nil, false
	}

	// Parse comma-separated linter list
	linters = strings.Split(matches[1], ",")
	for i, l := range linters {
		linters[i] = strings.ToLower(strings.TrimSpace(l))
	}

	return linters, true
}

const scopeguard = "scopeguard"

// containsScopeguard checks if "scopeguard" is in the linter list.
func containsScopeguard(linters []string) bool {
	for _, l := range linters {
		if strings.EqualFold(l, scopeguard) {
			return true
		}
	}

	return false
}

func createMessage(varNames []string, target ast.Node) (message string, related []analysis.RelatedInformation) {
	switch allNames := concatNames(varNames); target {
	case nil:
		format := "Variable %s is unused and can be removed"
		if len(varNames) > 1 {
			format = "Variables %s are unused and can be removed"
		}

		return fmt.Sprintf(format, allNames), nil

	default:
		scopeType := scopeTypeName(target)

		format := "Variable %s can be moved to tighter %s scope"
		if len(varNames) > 1 {
			format = "Variables %s can be moved to tighter %s scope"
		}

		return fmt.Sprintf(format, allNames, scopeType),
			[]analysis.RelatedInformation{{Pos: target.Pos(), Message: fmt.Sprintf("To this %s scope", scopeType)}}
	}
}

func collectNames(stmt ast.Node) []string {
	switch n := stmt.(type) {
	case *ast.AssignStmt:
		if n.Tok != token.DEFINE {
			break
		}

		varNames := make([]string, 0, len(n.Lhs))
		for id := range allAssigned(n) {
			varNames = append(varNames, id.Name)
		}

		return varNames

	case *ast.DeclStmt:
		decl, ok := n.Decl.(*ast.GenDecl)
		if !ok || decl.Tok != token.VAR {
			break
		}

		var varNames []string
		for id := range allDeclared(decl) {
			varNames = append(varNames, id.Name)
		}

		return varNames
	}

	return []string{"<unknown>"}
}

func concatNames(varNames []string) string {
	var allNames strings.Builder

	for i, name := range varNames {
		if i > 0 {
			var separator string
			if i == len(varNames)-1 {
				separator = " and "
			} else {
				separator = ", "
			}

			allNames.WriteString(separator)
		}

		allNames.WriteByte('\'')
		allNames.WriteString(name)
		allNames.WriteByte('\'')
	}

	return allNames.String()
}
