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
)

// ProcessDiagnostics generates and emits diagnostics for variables that can be moved to tighter scopes.
//
// This is the final phase of the analyzer pipeline. For each move target identified by the
// target phase, this function constructs a diagnostic message describing what can be moved
// and where, generates a suggested fix with text edits to perform the move (if possible) and
// reports the diagnostic to the analysis framework.
func ProcessDiagnostics(ctx context.Context, p *analysis.Pass, fdecl inspector.Cursor, diagnostics Diagnostics, option config.Behavior) {
	in := fdecl.Inspector()

	conservative := option.Enabled(config.Conservative)

	hadFixes := reportMoves(ctx, p, in, diagnostics.Moves, conservative)

	// Report nested assignments
	reportNestedAssigned(ctx, p, in, diagnostics.CurrentFile, diagnostics.Nested)

	// If hadFixes is true, variable renaming is suppressed. This is used to prevent conflicting
	// text edits when other fixes have already been applied in the same pass.
	rename := !hadFixes && option.Enabled(config.RenameVariables) && !diagnostics.Generated()

	// Report variables used after shadowed
	reportUsedAfterShadow(ctx, p, diagnostics.CurrentFile, fdecl, diagnostics.Shadows, rename)
}

func reportMoves(ctx context.Context, p *analysis.Pass, in *inspector.Inspector, moves []target.MoveTarget, conservative bool) bool {
	if len(moves) == 0 {
		return false
	}

	defer trace.StartRegion(ctx, "ReportMoves").End()

	hasFixes := false

	for _, move := range moves {
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

		if movable {
			if edits := createEdits(p, in, move); len(edits) > 0 {
				diagnostic.SuggestedFixes = []analysis.SuggestedFix{{Message: diagnostic.Message, TextEdits: edits}}
				hasFixes = true
			}
		}

		p.Report(diagnostic)
	}

	return hasFixes
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
		for id := range astutil.AllAssigned(n) {
			varNames = append(varNames, id.Name)
		}

		return varNames

	case *ast.DeclStmt:
		var varNames []string
		for id := range astutil.AllDeclared(n) {
			varNames = append(varNames, id.Name)
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
