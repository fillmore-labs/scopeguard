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
	"go/types"
	"runtime/trace"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/edge"
	"golang.org/x/tools/go/ast/inspector"

	"fillmore-labs.com/scopeguard/internal/astutil"
	"fillmore-labs.com/scopeguard/internal/config"
	"fillmore-labs.com/scopeguard/internal/report"
	"fillmore-labs.com/scopeguard/internal/scope"
	"fillmore-labs.com/scopeguard/internal/target"
	"fillmore-labs.com/scopeguard/internal/typeutil"
	"fillmore-labs.com/scopeguard/internal/usage"
)

// ErrResultMissing is returned when a required analyzer result is missing.
// This typically indicates a configuration error where the analyzer's
// Requires field is not properly set.
var ErrResultMissing = errors.New("analyzer result missing")

// Run executes the scopeguard analyzer's pipeline.
func (o *Options) Run(p *analysis.Pass) (any, error) {
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

	us := usage.New(p, scopes, o.Analyzers, o.Behaviors)

	ts := target.New(p, scopes, o.MaxLines, o.Behaviors)

	// Processed functions are returned as the result value so that
	//  callers can detect which function a diagnostic belongs to.
	var processed []Function

	// Loop over all files
	for c := range in.Root().Children() {
		file := c.Node().(*ast.File)

		currentFile := astutil.NewCurrentFile(p.Fset, file)
		if !currentFile.Valid() {
			astutil.InternalError(p, file, "File %s without valid info", file.Name.Name)
			continue
		}

		// Skip generated files
		if currentFile.Generated() && !o.Behaviors.Enabled(config.IncludeGenerated) {
			continue
		}

		// Skip files with nolint comment
		if astutil.CommentGroupHasNoLint(file.Doc) {
			continue
		}

		// Loop over all function and method declarations in this file
		for c := range c.Preorder((*ast.FuncDecl)(nil)) {
			f := c.Node().(*ast.FuncDecl)

			if f.Body == nil {
				continue
			}

			var fn typeutil.LocalFuncName
			if fun, ok := p.TypesInfo.Defs[f.Name].(*types.Func); ok {
				fn = typeutil.FuncNameOf(fun).LocalFuncName
			}

			// Check function n when we have a function filter
			if len(o.Functions) > 0 {
				if !slices.Contains(o.Functions, fn) {
					continue
				}
			} else {
				// Skip functions with nolint comment
				if astutil.CommentGroupHasNoLint(f.Doc) {
					continue
				}
			}

			processed = append(processed, Function{
				Pos:  f.Pos(),
				End:  f.End(),
				Name: fn,
			})

			body := c.ChildAt(edge.FuncDecl_Body, -1)

			// Stage 1: Collect all movable variable declarations and track variable uses
			usageData, usageDiagnostics := us.TrackUsage(ctx, body, f)

			var moves []target.MoveTarget

			// Stage 2: compute minimum safe scopes, select target nodes and resolve conflicts
			if usageData.HasScopeRanges() {
				// There are movable variable declarations
				moves = ts.SelectTargets(ctx, currentFile, body, usageData)
			}

			// Stage 3: Generate diagnostics with suggested fixes
			diagnostics := report.Diagnostics{
				CurrentFile: currentFile,
				Moves:       moves,
				Diagnostics: usageDiagnostics,
			}

			rename := o.Behaviors.Enabled(config.RenameVariables) && !diagnostics.Generated()
			renames := o.Renames[fn]

			diagnostics.Process(ctx, p, c, o.Filters, renames, rename)
		}
	}

	slices.SortFunc(processed, func(a, b Function) int { return int(a.Pos - b.Pos) })

	return Result{Processed: processed}, nil
}
