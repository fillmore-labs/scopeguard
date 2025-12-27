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

package analyze_test

import (
	"bytes"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/ast/edge"
	"golang.org/x/tools/go/ast/inspector"
)

const testpkg = "test"

// parseSource parses inline Go source code into an AST.
// The source is automatically wrapped in a function body: func _() { <src> }.
func parseSource(tb testing.TB, src string) (*token.FileSet, *ast.File) {
	tb.Helper()

	const (
		filename   = "test.go"
		header     = "package " + testpkg + "\n\nfunc _() {\n"
		suffix     = "\n}"
		wrapperLen = len(header) + len(suffix)
	)

	var srcFile bytes.Buffer
	srcFile.Grow(wrapperLen + len(src))

	srcFile.WriteString(header) // ignore error
	srcFile.WriteString(src)    // ignore error
	srcFile.WriteString(suffix) // ignore error

	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, filename, &srcFile, parser.SkipObjectResolution)
	if err != nil {
		tb.Fatalf("Failed to parse source %q: %v", src, err)
	}

	return fset, f
}

// checkSource creates a fully type-checked [types.Info] for unit testing.
// Use this when testing functions that require type information.
func checkSource(tb testing.TB, fset *token.FileSet, files []*ast.File) (*types.Package, *types.Info) {
	tb.Helper()

	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
	}

	conf := types.Config{Importer: importer.Default()}

	pkg, err := conf.Check(testpkg, fset, files, info)
	if err != nil {
		tb.Fatalf("failed to type Check source: %v", err)
	}

	return pkg, info
}

// firstBody returns an inspector cursor positioned at the first function declaration's body.
// This is useful for navigating the function body created by parseSource.
//
// Returns:
//   - body: Positioned at the FuncDecl's Body field
//   - typ:  The functions *ast.FuncType
func firstBody(t *testing.T, f *ast.File) (inspector.Cursor, *ast.FuncType) {
	t.Helper()

	c, ok := firstFuncDecl(f)
	if !ok {
		t.Fatal("Can't find function")
	}

	typ := c.Node().(*ast.FuncDecl).Type
	body := c.ChildAt(edge.FuncDecl_Body, -1)

	return body, typ
}

func firstFuncDecl(f *ast.File) (inspector.Cursor, bool) {
	root := inspector.New([]*ast.File{f}).Root()
	for c := range root.Preorder((*ast.FuncDecl)(nil)) {
		return c, true
	}

	return root, false
}
