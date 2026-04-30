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

// Diagnostics aggregates all analysis findings for the reporting stage.
type Diagnostics struct {
	astutil.CurrentFile
	Moves []target.MoveTarget
	usage.Diagnostics
}

// Process generates and emits diagnostics for variables that can be moved to tighter scopes.
//
// This is the final phase of the analyzer pipeline. For each move target identified by the
// target phase, this function constructs a diagnostic message describing what can be moved
// and where, generates a suggested fix with text edits to perform the move (if possible) and
// reports the diagnostic to the analysis framework.
func (d Diagnostics) Process(ctx context.Context, p *analysis.Pass, fdecl inspector.Cursor, filters config.Filters, renames map[string][]string, rename bool) {
	in := fdecl.Inspector()

	hadFixes := reportMoves(ctx, p, in, d.Moves, filters)

	// Nested assignments have no fixes, renames are always safe
	if !filters.Diagnostic().Has(config.Safe) {
		return
	}

	// Report nested assignments
	reportNestedAssigned(ctx, p, in, d.CurrentFile, d.Nested)

	// If hadFixes is true, variable renaming is suppressed. This is used to prevent conflicting
	// text edits when other fixes have already been applied in the same pass.
	rename = rename && !hadFixes && filters.Fix().Has(config.Safe)

	// Report variables used after shadowed
	reportUsedAfterShadow(ctx, p, d.CurrentFile, fdecl, d.Shadows, renames, rename)
}

// reportMoves emits diagnostics for move targets:
//
//   - Diagnostic creation: filtered by filters.Diagnostic().
//   - Fix generation: filtered by filters.Fix(), when a diagnostic exists.
func reportMoves(ctx context.Context, p *analysis.Pass, in *inspector.Inspector, moves []target.MoveTarget, filters config.Filters) bool {
	if len(moves) == 0 {
		return false
	}

	defer trace.StartRegion(ctx, "ReportMoves").End()

	hasFixes := false

	for _, move := range moves {
		safety := move.MoveStatus.Safety()
		if !filters.Diagnostic().Has(safety) {
			continue
		}

		node := move.Decl.Node(in)
		cat := move.MoveStatus.String()

		diagnostic := analysis.Diagnostic{
			Pos:      node.Pos(),
			End:      node.End(),
			Category: cat,
		}

		diagnostic.Message, diagnostic.Related = createMessage(in, move, cat)

		if filters.Fix().Has(safety) {
			if msg, edits := diagnostic.Message, createEdits(p, in, move); len(edits) > 0 {
				diagnostic.SuggestedFixes = []analysis.SuggestedFix{{Message: msg, TextEdits: edits}}
				hasFixes = true
			}
		}

		p.Report(diagnostic)
	}

	return hasFixes
}

// createMessage constructs the diagnostic message and related information.
func createMessage(in *inspector.Inspector, move target.MoveTarget, cat string) (message string, related []analysis.RelatedInformation) {
	switch move.TargetNode {
	case nil:
		format := "Variable %s is unused and can be removed (sg:%s)"
		if len(move.Unused) > 1 {
			format = "Variables %s are unused and can be removed (sg:%s)"
		}

		allNames := concatNames(move.Unused)

		return fmt.Sprintf(format, allNames, cat), nil

	default:
		varNames := declNames(in, move.MovableDecl)
		for _, absorbed := range move.AbsorbedDecls {
			varNames = append(varNames, declNames(in, absorbed)...)
		}

		var format string

		switch {
		case len(move.AbsorbedDecls) > 0:
			format = "Variables %s can be combined and moved to tighter %s scope (sg:%s)"
		case len(varNames) > 1:
			format = "Variables %s can be moved to tighter %s scope (sg:%s)"
		default:
			format = "Variable %s can be moved to tighter %s scope (sg:%s)"
		}

		allNames := concatNames(varNames)
		targetName := scope.Name(move.TargetNode)

		related := make([]analysis.RelatedInformation, 0, len(move.AbsorbedDecls)+1)

		for _, absorbed := range move.AbsorbedDecls {
			absorbedNode := absorbed.Decl.Node(in)
			related = append(related, analysis.RelatedInformation{
				Pos:     absorbedNode.Pos(),
				Message: "Combined with this declaration",
			})
		}

		related = append(related, analysis.RelatedInformation{
			Pos:     move.TargetNode.Pos(),
			Message: fmt.Sprintf("Moved to this %s scope", targetName),
		})

		return fmt.Sprintf(format, allNames, targetName, cat), related
	}
}

// declNames returns the declared names of `decl` minus any unused ones.
func declNames(in *inspector.Inspector, decl target.MovableDecl) []string {
	names := collectNames(decl.Decl.Node(in))

	if len(decl.Unused) > 0 {
		names = slices.DeleteFunc(names, func(name string) bool { return slices.Contains(decl.Unused, name) })
	}

	return names
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
