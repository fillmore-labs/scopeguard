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

package usage

import (
	"context"
	"go/ast"
	"go/types"
	"runtime/trace"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"

	"fillmore-labs.com/scopeguard/internal/astutil"
	"fillmore-labs.com/scopeguard/internal/config"
	"fillmore-labs.com/scopeguard/internal/scope"
	"fillmore-labs.com/scopeguard/internal/usage/check"
)

// Stage configures and runs the usage analysis stage.
// It captures the analysis pass context and configuration to instantiate a usageCollector.
type Stage struct {
	*analysis.Pass
	scope.UsageScope
	Analyzers config.BitMask[config.AnalyzerFlags]
}

// TrackUsage collects variable declarations and tracks their usages to determine the minimum scope.
func (us Stage) TrackUsage(ctx context.Context, body inspector.Cursor, f *ast.FuncDecl) (Result, Diagnostics) {
	defer trace.StartRegion(ctx, "Usage").End()

	uc := us.newUsageCollector()

	uc.handleFunc(body, f.Recv, f.Type)
	uc.inspectBody(body, f.Type.Results)

	return uc.result()
}

// newUsageCollector creates a new usage collector for analyzing a function body.
func (us Stage) newUsageCollector() collector {
	var scopeRanges map[astutil.NodeIndex]ScopeRange

	if us.Analyzers.Enabled(config.ScopeAnalyzer) {
		scopeRanges = make(map[astutil.NodeIndex]ScopeRange)
	}

	return collector{
		Pass:          us.Pass,
		UsageScope:    us.UsageScope,
		ShadowChecker: check.NewShadowChecker(us.Analyzers.Enabled(config.ShadowAnalyzer)),
		NestedChecker: check.NewNestedChecker(us.Analyzers.Enabled(config.NestedAssignAnalyzer)),
		scopeRanges:   scopeRanges,
		current:       make(map[*types.Var]declUsage),
		usages:        make(map[*types.Var][]NodeUsage),
	}
}
