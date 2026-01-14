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

package tracker_test

import (
	"fmt"
	"go/ast"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	. "fillmore-labs.com/scopeguard/internal/reachability/tracker"
)

func TestCantReturn(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()

	testAnalyzer := &analysis.Analyzer{
		Name:     "cantreturnanalyzer",
		Doc:      "test cantreturn",
		Run:      crrun,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}

	analysistest.Run(t, testdata, testAnalyzer, "./cantreturn")
}

func crrun(p *analysis.Pass) (any, error) {
	in, ok := p.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("result of %s missing", inspect.Analyzer.Name)
	}

	types, visit := []ast.Node{(*ast.File)(nil), (*ast.ExprStmt)(nil)}, crpass{p}.inspect
	in.Nodes(types, visit)

	return any(nil), nil
}

type crpass struct{ *analysis.Pass }

func (p crpass) inspect(n ast.Node, push bool) (proceed bool) {
	if !push {
		return true
	}

	switch n := n.(type) {
	case *ast.File:
		if ast.IsGenerated(n) {
			return false
		}

	case *ast.ExprStmt:
		expr, ok := n.X.(*ast.CallExpr)
		if !ok || !CantReturn(p.TypesInfo, expr) {
			break
		}

		p.Report(analysis.Diagnostic{
			Pos:     expr.Pos(),
			End:     expr.End(),
			Message: "Can't return",
		})
	}

	return true
}
