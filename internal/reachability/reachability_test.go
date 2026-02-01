// Copyright 2026 Oliver Eikemeier. All Rights Reserved.
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

package reachability_test

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	. "fillmore-labs.com/scopeguard/internal/reachability"
)

func TestReachable(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()

	testAnalyzer := &analysis.Analyzer{
		Name: "reachabilitytest",
		Doc:  "test reachability",
		Run: func(p *analysis.Pass) (any, error) {
			return reachability(t.Context(), p)
		},
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}

	analysistest.Run(t, testdata, testAnalyzer, "./graph")
}

func reachability(ctx context.Context, p *analysis.Pass) (any, error) {
	in, ok := p.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("result of %s missing", inspect.Analyzer.Name)
	}

	generated := false

	for c := range in.Root().Preorder((*ast.File)(nil), (*ast.FuncDecl)(nil)) {
		switch n := c.Node().(type) {
		case *ast.File:
			generated = ast.IsGenerated(n)

		case *ast.FuncDecl:
			if generated {
				continue
			}

			// Find the first declaration of "from" and first use of "to"
			fpos, tpos := findFromTo(p.TypesInfo, c)

			if !fpos.IsValid() {
				p.Report(analysis.Diagnostic{
					Pos:     n.Pos(),
					Message: "Can't find from",
				})

				continue
			}

			if !tpos.IsValid() {
				p.Report(analysis.Diagnostic{
					Pos:     n.Pos(),
					Message: "Can't find to",
				})

				continue
			}

			forwardOnly := !strings.HasSuffix(n.Name.Name, "L")
			graph := NewGraph(ctx, p.TypesInfo, n.Recv, n.Type, n.Body, forwardOnly)

			if reachable, ok := graph.Reachable(fpos, tpos); ok {
				message := "unreachable"
				if reachable {
					message = "is reachable"
				}

				p.Report(analysis.Diagnostic{
					Pos:     tpos,
					Message: message,
				})

				continue
			}

			p.Report(analysis.Diagnostic{
				Pos:     n.Pos(),
				Message: "Can't find from, to",
			})

		default:
			panic(fmt.Sprintf("Unexpected node type %T", n))
		}
	}

	return any(nil), nil
}

func findFromTo(info *types.Info, c inspector.Cursor) (fpos, tpos token.Pos) {
	for id := range c.Preorder((*ast.Ident)(nil)) {
		switch id := id.Node().(*ast.Ident); id.Name {
		case "from":
			if _, ok := info.Defs[id]; ok && !fpos.IsValid() {
				fpos = id.NamePos
			}
		case "to":
			if _, ok := info.Uses[id]; ok && !tpos.IsValid() {
				tpos = id.NamePos
			}
		}
	}

	return fpos, tpos
}
