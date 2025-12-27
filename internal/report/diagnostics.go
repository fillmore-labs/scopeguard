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
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"runtime/trace"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"

	"fillmore-labs.com/scopeguard/internal/astutil"
	"fillmore-labs.com/scopeguard/internal/config"
	"fillmore-labs.com/scopeguard/internal/scope"
	"fillmore-labs.com/scopeguard/internal/target"
	"fillmore-labs.com/scopeguard/internal/usage"
)

// ProcessDiagnostics generates and emits diagnostics for variables that can be moved to tighter scopes.
//
// This is the final phase of the analyzer pipeline. For each move target identified by the
// target phase, this function constructs a diagnostic message describing what can be moved
// and where, generates a suggested fix with text edits to perform the move (if possible) and
// reports the diagnostic to the analysis framework.
func ProcessDiagnostics(ctx context.Context, p *analysis.Pass, currentFile astutil.CurrentFile, fdecl inspector.Cursor, diagnostics Diagnostics, option config.BitMask[config.Config]) {
	defer trace.StartRegion(ctx, "Report").End()

	in := fdecl.Inspector()

	// Report nested assignments
	reportNestedAssigned(ctx, p, in, currentFile, diagnostics.Nested)

	// Report variables used after shadowed
	rename := option.Enabled(config.RenameVariables) && !currentFile.Generated()
	hadFixes := reportUsedAfterShadow(ctx, p, currentFile, fdecl, diagnostics.Shadows, rename)

	if len(diagnostics.Moves) == 0 {
		return
	}

	conservative := option.Enabled(config.Conservative)

	for _, move := range diagnostics.Moves {
		movable := move.Status.Movable()
		if conservative && !movable {
			continue
		}

		c := move.Decl.Cursor(in)
		node := c.Node()

		diagnostic := analysis.Diagnostic{
			Pos: node.Pos(),
			End: node.End(),
		}

		diagnostic.Message, diagnostic.Related = createMessage(in, move)

		if movable && !hadFixes {
			// If hadFixes is true, suggested fixes are suppressed. This is used to prevent conflicting
			// text edits when other fixes (like variable renaming) have already been applied in the same pass.
			if edits := createEdits(p, in, move); len(edits) > 0 {
				diagnostic.SuggestedFixes = []analysis.SuggestedFix{{Message: diagnostic.Message, TextEdits: edits}}
			}
		}

		p.Report(diagnostic)
	}
}

// reportNestedAssigned emits diagnostics for nested assigns of variables.
func reportNestedAssigned(ctx context.Context, p *analysis.Pass, in *inspector.Inspector, currentFile astutil.CurrentFile, nested []usage.NestedAssign) {
	defer trace.StartRegion(ctx, "ReportNestedAssigned").End()

	for _, assignment := range nested {
		if currentFile.NoLintComment(assignment.Ident.Pos()) {
			continue
		}

		stmt := assignment.Asgn.Node(in)

		p.Report(analysis.Diagnostic{
			Pos:     assignment.Ident.Pos(),
			End:     assignment.Ident.End(),
			Message: fmt.Sprintf("Nested reassignment of variable '%s' (sg:nst)", assignment.Ident.Name),
			Related: []analysis.RelatedInformation{{
				Pos:     stmt.Pos(),
				End:     stmt.End(),
				Message: "Inside this assign statement",
			}},
		})
	}
}

// reportUsedAfterShadow emits diagnostics for variables used after previously shadowed.
func reportUsedAfterShadow(ctx context.Context, p *analysis.Pass, currentFile astutil.CurrentFile, fdecl inspector.Cursor, shadows []usage.ShadowUse, rename bool) bool {
	defer trace.StartRegion(ctx, "ReportShadowed").End()

	var renamer *Renamer
	if rename {
		renamer = NewRenamer()
	}

	hadFixes := false

	in := fdecl.Inspector()

	for _, shadowed := range shadows {
		use := shadowed.Use.Node(in)
		if currentFile.NoLintComment(use.Pos()) {
			continue
		}

		suggestedFixes := renamer.Renames(p.TypesInfo, fdecl, shadowed.Var)

		if len(suggestedFixes) > 0 {
			hadFixes = true
		}

		name, decl := shadowed.Var.Name(), shadowed.Decl.Node(in)
		p.Report(analysis.Diagnostic{
			Pos:            use.Pos(),
			End:            use.End(),
			Message:        fmt.Sprintf("Identifier '%s' used after previously shadowed (sg:uas)", name),
			Related:        []analysis.RelatedInformation{{Pos: decl.Pos(), End: decl.Pos(), Message: "After this declaration"}},
			SuggestedFixes: suggestedFixes,
		})
	}

	return hadFixes
}

// createMessage constructs the diagnostic message and related information.
func createMessage(in *inspector.Inspector, move target.MoveTarget) (message string, related []analysis.RelatedInformation) {
	switch move.TargetNode {
	case nil:
		format := "Variable %s is unused and can be removed (sg:%s)"
		if len(move.Unused) > 1 {
			format = "Variables %s are unused and can be removed (sg:%s)"
		}

		allNames := concatNames(move.Unused)

		return fmt.Sprintf(format, allNames, move.Status), nil

	default:
		node := move.Decl.Node(in)
		varNames := collectNames(node)

		if len(move.Unused) > 0 {
			varNames = slices.DeleteFunc(varNames, func(name string) bool { return slices.Contains(move.Unused, name) })
		}

		format := "Variable %s can be moved to tighter %s scope (sg:%s)"
		if len(varNames) > 1 {
			format = "Variables %s can be moved to tighter %s scope (sg:%s)"
		}

		allNames := concatNames(varNames)
		targetName := scope.Name(move.TargetNode)

		return fmt.Sprintf(format, allNames, targetName, move.Status),
			[]analysis.RelatedInformation{{Pos: move.TargetNode.Pos(), Message: fmt.Sprintf("To this %s scope", targetName)}}
	}
}

// collectNames extracts variable names from a declaration statement.
func collectNames(stmt ast.Node) []string {
	switch n := stmt.(type) {
	case *ast.AssignStmt:
		if n.Tok != token.DEFINE {
			break
		}

		varNames := make([]string, 0, len(n.Lhs))
		for name := range astutil.AllAssignedNames(n) {
			varNames = append(varNames, name)
		}

		return varNames

	case *ast.DeclStmt:
		var varNames []string
		for name := range astutil.AllDeclaredNames(n) {
			varNames = append(varNames, name)
		}

		return varNames
	}

	return []string{"<unknown>"}
}

// concatNames formats a list of variable names into a human-readable string (e.g., "'a', 'b' and 'c'").
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

			allNames.WriteString(separator) // ignore error
		}

		allNames.WriteByte('\'')   // ignore error
		allNames.WriteString(name) // ignore error
		allNames.WriteByte('\'')   // ignore error
	}

	return allNames.String()
}
