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
)

// run executes the scopeguard analyzer's pipeline.
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

	g := make(generatedChecker)
	// Loop over all function and method declarations
	for c := range in.Root().Preorder((*ast.FuncDecl)(nil)) {
		// One function declaration is always in a single file
		file := enclosingFile(c)

		generated := g.isGenerated(file)
		if generated && !o.Generated {
			continue
		}

		// Stage 1: Collect all movable variable declarations and track variable uses
		usageScopes := p.usage(ctx, scopes, c)
		if usageScopes.scopeRanges == nil {
			// No movable variable declarations
			continue
		}

		// Stage 2: compute minimum safe scopes, select target nodes and resolve conflicts
		targets := p.target(ctx, in, file, generated, scopes, usageScopes, o.MaxLines)

		// Stage 3: Generate diagnostics with suggested fixes
		p.report(ctx, in, targets)
	}

	return nil, nil
}
