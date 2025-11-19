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
	"runtime/trace"

	"golang.org/x/tools/go/analysis"
)

// run executes the scopeguard analyzer's four-stage pipeline.
//
// Pipeline stages:
//
//  1. declarations: Collects all := and var declarations in the analyzed code
//     Returns: varDecls map (variable -> declaration info)
//
//  2. usage: Tracks where each variable is used and computes the tightest scope
//     that contains all uses (the lowest common ancestor in scope tree)
//     Mutates: decls[v].targetScope with computed minimum scope
//
//  3. target: Selects concrete AST nodes to move declarations to, applying safety
//     constraints and resolving Init field conflicts
//     Returns: []moveTarget (the sorted list of moves to perform)
//
//  4. report: Generates diagnostics with suggested fixes for each move target
//     Side effect: Calls pass.Report() to emit diagnostics
//
// The analyzer uses runtime/trace for performance profiling. Use:
//
//	go test -trace=trace.out ./analyzer/...
//	go tool trace trace.out
//
// to analyze performance characteristics.
func (o *Options) run(pass *analysis.Pass) (any, error) {
	ctx := context.Background()

	ctx, task := trace.NewTask(ctx, "scopeguard")
	defer task.End()

	p := newPass(pass)

	in, err := p.inspector()
	if err != nil {
		return nil, err
	}

	// Build inverted scope->node map for bidirectional AST/scope navigation
	scopes := p.scopeAnalyzer()

	// Stage 1: Collect all movable variable declarations
	decls, err := p.declarations(ctx, in, o.Generated)
	if err != nil {
		return nil, err
	}

	// Stage 2: Track variable uses and compute minimum safe scopes
	usageScopes, err := p.usage(ctx, in, scopes, decls)
	if err != nil {
		return nil, err
	}

	// Stage 3: Select target nodes and resolve conflicts
	targets, err := p.target(ctx, in, scopes, usageScopes)
	if err != nil {
		return nil, err
	}

	// Stage 4: Generate diagnostics with suggested fixes
	if err := p.report(ctx, in, targets, o.Generated); err != nil {
		return nil, err
	}

	return nil, nil
}
