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
	"go/ast"
	"go/types"
	"runtime/trace"

	"golang.org/x/tools/go/ast/inspector"
)

// UsageAnalyzer configures and runs the usage analysis stage.
// It captures the analysis pass context and configuration to instantiate a usageCollector.
type UsageAnalyzer struct {
	Pass
	ScopeIndex
	Analyzers BitMask[Analyzer]
}

// Analyze collects variable declarations and tracks their usages to determine the minimum scope.
func (ua UsageAnalyzer) Analyze(ctx context.Context, body inspector.Cursor, f *ast.FuncDecl) (UsageData, UsageDiagnostics) {
	defer trace.StartRegion(ctx, "Usage").End()

	uc := ua.newUsageCollector()

	uc.inspectBody(body, f.Recv, f.Type)

	return uc.result()
}

// newUsageCollector creates a new usage collector for analyzing a function body.
func (ua UsageAnalyzer) newUsageCollector() usageCollector {
	var scopeRanges map[NodeIndex]ScopeRange

	if ua.Analyzers.Enabled(ScopeAnalyzer) {
		scopeRanges = make(map[NodeIndex]ScopeRange)
	}

	return usageCollector{
		Pass:          ua.Pass,
		ScopeIndex:    ua.ScopeIndex,
		ShadowChecker: NewShadowChecker(ua.Analyzers.Enabled(ShadowAnalyzer)),
		NestedChecker: NewNestedChecker(ua.Analyzers.Enabled(NestedAssignAnalyzer)),
		scopeRanges:   scopeRanges,
		current:       make(map[*types.Var]declUsage),
		usages:        make(map[*types.Var][]NodeUsage),
	}
}
