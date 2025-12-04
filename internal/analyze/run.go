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
	"go/ast"
	"runtime/trace"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/edge"
	"golang.org/x/tools/go/ast/inspector"

	"fillmore-labs.com/scopeguard/analyzer/level"
)

// run executes the scopeguard analyzer's pipeline.
func (o *Options) run(ap *analysis.Pass) (any, error) {
	ctx := context.Background()

	ctx, task := trace.NewTask(ctx, "ScopeGuard")
	defer task.End()

	p := Pass{Pass: ap}

	in, err := p.Inspector()
	if err != nil {
		return nil, err
	}

	// Build inverted scope->node map for bidirectional AST/scope navigation
	scopes := NewScopeAnalyzer(p)

	// Remember the current file over all functions declared in it
	var currentFile CurrentFile
	// Loop over all function and method declarations
	in.Root().Inspect(
		[]ast.Node{(*ast.File)(nil), (*ast.FuncDecl)(nil)},
		func(c inspector.Cursor) bool {
			switch n := c.Node().(type) {
			case *ast.File:
				currentFile = NewCurrentFile(p.Fset, n)

				return o.Generated || !currentFile.Generated()

			case *ast.FuncDecl:
				if n.Body == nil {
					return false
				}

				if !currentFile.Valid() {
					p.ReportInternalError(n, "Function declaration %s without file info", n.Name.Name)

					return false
				}

				// Stage 1: Collect all movable variable declarations and track variable uses
				body, funcType := c.ChildAt(edge.FuncDecl_Body, -1), n.Type
				usageScopes, usedAfterShadow, nestedAssigned := Usage(ctx, p, scopes, body, funcType, o.ScopeLevel, o.ShadowLevel, o.NestedAssign)

				// Report nested assignments
				nestedAssigned.Report(ctx, p.Pass, currentFile)

				// Report variables used after shadowed
				usedAfterShadow.Report(ctx, p.Pass, in, currentFile)

				if len(usageScopes.ScopeRanges) == 0 {
					return false // No movable variable declarations
				}

				conservative := o.ScopeLevel == level.ScopeConservative

				// Stage 2: compute minimum safe scopes, select target nodes and resolve conflicts
				targets := Targets(ctx, p, in, TargetOptions{
					ScopeAnalyzer: scopes,
					CurrentFile:   currentFile,
					UsageResult:   usageScopes,
					MaxLines:      o.MaxLines,
					Conservative:  conservative,
				})

				// Stage 3: Generate diagnostics with suggested fixes
				Report(ctx, p, in, targets, conservative)

				return true

			default:
				p.ReportInternalError(n, "Unexpected node type: %T", n)

				return false
			}
		},
	)

	return nil, nil
}
