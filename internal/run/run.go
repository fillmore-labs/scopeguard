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

package run

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

// Run executes the scopeguard analyzer's pipeline.
func (r *Options) Run(p *analysis.Pass) (any, error) {
	// Retrieves the [inspector.Inspector] from the pass results.
	in, ok := p.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("scopeguard: %s %w", inspect.Analyzer.Name, ErrResultMissing)
	}

	ctx := context.Background()

	ctx, task := trace.NewTask(ctx, "ScopeGuard")
	defer task.End()

	trace.Log(ctx, "package", p.Pkg.Path())

	// Build inverted scope->node map for bidirectional AST/scope navigation
	scopes := scope.NewIndex(p.TypesInfo)

	us := usage.New(p, scopes, r.Analyzers, r.Behavior)

	ts := target.New(p, scopes, r.MaxLines, r.Behavior)

	// Remember the current file over all functions declared in it
	var currentFile astutil.CurrentFile

	// Loop over all files
	for f := range in.Root().Children() {
		file := f.Node().(*ast.File)

		currentFile = astutil.NewCurrentFile(p.Fset, file)
		if !currentFile.Valid() {
			astutil.InternalError(p, file, "File %s without valid info", file.Name.Name)

			continue
		}

		// Skip generated files
		if currentFile.Generated() && !r.Behavior.Enabled(config.IncludeGenerated) {
			continue
		}

		// Skip files with nolint comment
		if file.Doc != nil && astutil.CommentHasNoLint(file.Doc.List[len(file.Doc.List)-1]) {
			continue
		}

		// Loop over all function and method declarations in this file
		for c := range f.Preorder((*ast.FuncDecl)(nil)) {
			fun := c.Node().(*ast.FuncDecl)

			if fun.Body == nil {
				continue
			}

			// Skip functions with nolint comment
			if fun.Doc != nil && astutil.CommentHasNoLint(fun.Doc.List[len(fun.Doc.List)-1]) {
				continue
			}

			body := c.ChildAt(edge.FuncDecl_Body, -1)

			// Stage 1: Collect all movable variable declarations and track variable uses
			usageData, usageDiagnostics := us.TrackUsage(ctx, body, fun)

			var moves []target.MoveTarget

			// Stage 2: compute minimum safe scopes, select target nodes and resolve conflicts
			if usageData.HasScopeRanges() {
				// There are movable variable declarations
				moves = ts.SelectTargets(ctx, currentFile, body, usageData)
			}

			diagnostics := report.Diagnostics{
				CurrentFile: currentFile,
				Moves:       moves,
				Diagnostics: usageDiagnostics,
			}

			// Stage 3: Generate diagnostics with suggested fixes
			report.ProcessDiagnostics(ctx, p, c, diagnostics, r.Behavior)
		}
	}

	return nil, nil
}
