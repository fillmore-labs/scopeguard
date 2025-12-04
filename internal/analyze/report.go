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
	"runtime/trace"
	"slices"
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
func (p pass) report(ctx context.Context, in *inspector.Inspector, targets targetResult) {
	defer trace.StartRegion(ctx, "report").End()

	for _, move := range targets.move {
		c := in.At(move.decl)
		node := c.Node()

		diagnostic := analysis.Diagnostic{
			Pos: node.Pos(),
			End: node.End(),
		}

		diagnostic.Message, diagnostic.Related = createMessage(node, move.targetNode, move.unused)

		if !move.dontFix {
			if edits := p.createEdits(c, move.targetNode, move.unused); len(edits) > 0 {
				diagnostic.SuggestedFixes = []analysis.SuggestedFix{{Message: diagnostic.Message, TextEdits: edits}}
			}
		}

		p.Report(diagnostic)
	}
}

// createMessage constructs the diagnostic message and related information.
func createMessage(node, target ast.Node, unused []string) (message string, related []analysis.RelatedInformation) {
	switch target {
	case nil:
		format := "Variable %s is unused and can be removed"
		if len(unused) > 1 {
			format = "Variables %s are unused and can be removed"
		}

		allNames := concatNames(unused)

		return fmt.Sprintf(format, allNames), nil

	default:
		varNames := collectNames(node)
		if len(unused) > 0 {
			varNames = slices.DeleteFunc(varNames, func(name string) bool { return slices.Contains(unused, name) })
		}

		format := "Variable %s can be moved to tighter %s scope"
		if len(varNames) > 1 {
			format = "Variables %s can be moved to tighter %s scope"
		}

		allNames := concatNames(varNames)
		scopeType := scopeTypeName(target)

		return fmt.Sprintf(format, allNames, scopeType),
			[]analysis.RelatedInformation{{Pos: target.Pos(), Message: fmt.Sprintf("To this %s scope", scopeType)}}
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

			allNames.WriteString(separator)
		}

		allNames.WriteByte('\'')
		allNames.WriteString(name)
		allNames.WriteByte('\'')
	}

	return allNames.String()
}
