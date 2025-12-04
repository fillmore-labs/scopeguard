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

	"golang.org/x/tools/go/ast/edge"
	"golang.org/x/tools/go/ast/inspector"
)

// usage collects variable declarations and tracks their usages to determine the minimum scope.
func (p pass) usage(ctx context.Context, scopes scopeAnalyzer, f inspector.Cursor) usageResult {
	defer trace.StartRegion(ctx, "usage").End()

	n, ok := f.Node().(*ast.FuncDecl)
	if !ok || n.Body == nil {
		return usageResult{}
	}

	body, results := f.ChildAt(edge.FuncDecl_Body, -1), n.Type.Results

	uc := p.newUsageCollector(scopes)
	uc.inspectBody(body, results)

	return uc.result()
}
