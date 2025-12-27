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

package analyze

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"runtime/trace"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/ast/inspector"
)

// Report generates and emits diagnostics for variables that can be moved to tighter scopes.
//
// This is the final phase of the analyzer pipeline. For each move target identified by the
// target phase, this function constructs a diagnostic message describing what can be moved
// and where, generates a suggested fix with text edits to perform the move (if possible) and
// reports the diagnostic to the analysis framework.
func Report(ctx context.Context, p Pass, in *inspector.Inspector, c inspector.Cursor, currentFile CurrentFile, result ReportData, option BitMask[Config]) {
	defer trace.StartRegion(ctx, "Report").End()

	// Report nested assignments
	result.Nested.Report(ctx, p.Pass, in, currentFile)

	// Report variables used after shadowed
	rename := option.Enabled(RenameVars) && !currentFile.Generated()
	hadFixes := result.Shadows.Report(ctx, p.Pass, c, currentFile, rename)

	if len(result.Moves) == 0 {
		return
	}

	conservative := option.Enabled(Conservative)

	for _, move := range result.Moves {
		movable := move.Status.Movable()
		if conservative && !movable {
			continue
		}

		c := in.At(move.Decl)
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

// createMessage constructs the diagnostic message and related information.
func createMessage(in *inspector.Inspector, move MoveTarget) (message string, related []analysis.RelatedInformation) {
	switch move.TargetNode {
	case nil:
		format := "Variable %s is unused and can be removed (sg:%s)"
		if len(move.Unused) > 1 {
			format = "Variables %s are unused and can be removed (sg:%s)"
		}

		allNames := concatNames(move.Unused)

		return fmt.Sprintf(format, allNames, move.Status), nil

	default:
		node := in.At(move.Decl).Node()
		varNames := collectNames(node)

		if len(move.Unused) > 0 {
			varNames = slices.DeleteFunc(varNames, func(name string) bool { return slices.Contains(move.Unused, name) })
		}

		format := "Variable %s can be moved to tighter %s scope (sg:%s)"
		if len(varNames) > 1 {
			format = "Variables %s can be moved to tighter %s scope (sg:%s)"
		}

		allNames := concatNames(varNames)
		targetName := ScopeName(move.TargetNode)

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
		for id := range allAssigned(n) {
			varNames = append(varNames, id.Name)
		}

		return varNames

	case *ast.DeclStmt:
		var varNames []string
		for id := range allDeclared(n) {
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

// ScopeName returns a human-readable name for the scope type.
func ScopeName(node ast.Node) string {
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
		return astutil.NodeDescription(node)
		// keep-sorted end
	}
}
