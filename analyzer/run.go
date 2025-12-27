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

package analyzer

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

	"fillmore-labs.com/scopeguard/internal/astutil"
	"fillmore-labs.com/scopeguard/internal/config"
	"fillmore-labs.com/scopeguard/internal/report"
	"fillmore-labs.com/scopeguard/internal/scope"
	"fillmore-labs.com/scopeguard/internal/target"
	"fillmore-labs.com/scopeguard/internal/usage"
)

// ErrResultMissing is returned when a required analyzer result is missing.
// This typically indicates a configuration error where the analyzer's
// Requires field is not properly set.
var ErrResultMissing = errors.New("analyzer result missing")

// run executes the scopeguard analyzer's pipeline.
func (r *runOptions) run(p *analysis.Pass) (any, error) {
	// Retrieves the [inspector.Inspector] from the pass results.
	in, ok := p.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("scopeguard: %s %w", inspect.Analyzer.Name, ErrResultMissing)
	}

	ctx := context.Background()

	ctx, task := trace.NewTask(ctx, "ScopeGuard")
	defer task.End()

	// Build inverted scope->node map for bidirectional AST/scope navigation
	scopes := scope.NewIndex(p.TypesInfo.Scopes)

	us := usage.Stage{
		Pass:       p,
		UsageScope: scope.NewUsageScope(scopes),
		Analyzers:  r.analyzers,
	}

	ts := target.Stage{
		Pass:         p,
		TargetScope:  scope.NewTargetScope(scopes),
		MaxLines:     r.maxLines,
		Conservative: r.behavior.Enabled(config.Conservative),
		Combine:      r.behavior.Enabled(config.CombineDeclarations),
	}

	// Remember the current file over all functions declared in it
	var currentFile astutil.CurrentFile

	// Loop over all function and method declarations
	root, types := in.Root(), []ast.Node{
		(*ast.File)(nil),
		(*ast.FuncDecl)(nil),
	}

	// Loop over all function and method declarations
	root.Inspect(types, func(i inspector.Cursor) bool {
		switch node := i.Node().(type) {
		case *ast.File:
			currentFile = astutil.NewCurrentFile(p.Fset, node)
			descend := r.behavior.Enabled(config.IncludeGenerated) || !currentFile.Generated()

			return descend

		case *ast.FuncDecl:
			if node.Body == nil {
				return false
			}

			if !currentFile.Valid() {
				astutil.InternalError(p, node, "Function declaration %s without file info", node.Name.Name)

				return false
			}

			// Skip functions with nolint comment
			if node.Doc != nil && astutil.CommentHasNoLint(node.Doc.List[len(node.Doc.List)-1]) {
				return false
			}

			body := i.ChildAt(edge.FuncDecl_Body, -1)

			// Stage 1: Collect all movable variable declarations and track variable uses
			usageData, usageDiagnostics := us.TrackUsage(ctx, body, node)

			var moves []target.MoveTarget

			// Stage 2: compute minimum safe scopes, select target nodes and resolve conflicts
			if usageData.HasScopeRanges() {
				// There are movable variable declarations
				moves = ts.SelectTargets(ctx, currentFile, body, usageData)
			}

			diagnostics := report.Diagnostics{
				Moves:       moves,
				Diagnostics: usageDiagnostics,
			}

			// Stage 3: Generate diagnostics with suggested fixes
			report.ProcessDiagnostics(ctx, p, currentFile, i, diagnostics, r.behavior)

			return true

		default:
			astutil.InternalError(p, node, "Unexpected node type: %T", node)

			return false
		}
	})

	return nil, nil
}
