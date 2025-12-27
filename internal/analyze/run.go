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
	"errors"
	"fmt"
	"go/ast"
	"runtime/trace"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/edge"
	"golang.org/x/tools/go/ast/inspector"
)

// ErrResultMissing is returned when a required analyzer result is missing.
// This typically indicates a configuration error where the analyzer's
// Requires field is not properly set.
var ErrResultMissing = errors.New("analyzer result missing")

// run executes the scopeguard analyzer's pipeline.
func (o *Options) run(ap *analysis.Pass) (any, error) {
	// Retrieves the [inspector.Inspector] from the pass results.
	in, ok := ap.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("scopeguard: %s %w", inspect.Analyzer.Name, ErrResultMissing)
	}

	ctx := context.Background()

	ctx, task := trace.NewTask(ctx, "ScopeGuard")
	defer task.End()

	p := Pass{Pass: ap}

	// Build inverted scope->node map for bidirectional AST/scope navigation
	scopes := NewScopeIndex(p.TypesInfo.Scopes)

	ua := UsageAnalyzer{
		Pass:       p,
		ScopeIndex: scopes,
		Analyzers:  o.Analyzers,
	}

	ta := TargetAnalyzer{
		Pass:          p,
		TargetScope:   NewTargetScope(scopes),
		SafetyChecker: NewSafetyChecker(p.TypesInfo),
		MaxLines:      o.MaxLines,
		Conservative:  o.Behavior.Enabled(Conservative),
		Combine:       o.Behavior.Enabled(CombineDecls),
	}

	// Remember the current file over all functions declared in it
	var currentFile CurrentFile

	// Loop over all function and method declarations
	root, types := in.Root(), []ast.Node{
		(*ast.File)(nil),
		(*ast.FuncDecl)(nil),
	}

	// Loop over all function and method declarations
	root.Inspect(types, func(c inspector.Cursor) bool {
		switch node := c.Node().(type) {
		case *ast.File:
			currentFile = NewCurrentFile(p.Fset, node)
			descend := o.Behavior.Enabled(IncludeGenerated) || !currentFile.Generated()

			return descend

		case *ast.FuncDecl:
			if node.Body == nil {
				return false
			}

			if !currentFile.Valid() {
				p.ReportInternalError(node, "Function declaration %s without file info", node.Name.Name)

				return false
			}

			// Skip functions with nolint comment
			if node.Doc != nil && CommentHasNoLint(node.Doc.List[len(node.Doc.List)-1]) {
				return false
			}

			body := c.ChildAt(edge.FuncDecl_Body, -1)

			// Stage 1: Collect all movable variable declarations and track variable uses
			usageData, usageDiagnostics := ua.Analyze(ctx, body, node)

			var moves []MoveTarget

			// Stage 2: compute minimum safe scopes, select target nodes and resolve conflicts
			if len(usageData.ScopeRanges) > 0 {
				// There are movable variable declarations
				moves = ta.Analyze(ctx, currentFile, body, usageData)
			}

			// Stage 3: Generate diagnostics with suggested fixes
			reportData := ReportData{
				Moves:            moves,
				UsageDiagnostics: usageDiagnostics,
			}
			Report(ctx, p, in, c, currentFile, reportData, o.Behavior)

			return true

		default:
			p.ReportInternalError(node, "Unexpected node type: %T", node)

			return false
		}
	})

	return nil, nil
}
