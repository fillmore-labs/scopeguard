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

// Package testsource provides utilities for parsing and analyzing Go source code in tests.
//
// It is designed to simplify testing of the scopeguard analyzer by handling common
// boilerplate code for parsing and type-checking Go source fragments.
package testsource

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

// Parse parses a Go source code fragment into an AST.
// The provided source `src` is automatically wrapped in a function body `func _() { ... }`
// within a package `test`. This allows testing statement-level code fragments without
// manually constructing the surrounding package and function scaffolding.
//
// Call [Check] on the result when type information is needed.
//
// Returns:
//   - *token.FileSet: The file set containing the single source file.
//   - *ast.File: The parsed AST of the source file.
//   - *ast.FuncDecl: The function declaration wrapping the source code.
//   - inspector.Cursor: A cursor positioned at the wrapper function's Body field.
func Parse(tb testing.TB, src string) (fset *token.FileSet, f *ast.File, fn *ast.FuncDecl, body inspector.Cursor) {
	tb.Helper()

	const filename = "test.go"

	fset = token.NewFileSet()
	srcFile := wrapSource(src)

	f, err := parser.ParseFile(fset, filename, srcFile, parser.SkipObjectResolution)
	if err != nil {
		tb.Fatalf("Failed to parse source %q: %v", src, err)
	}

	fn, body = firstFuncDecl(f)
	if fn == nil {
		tb.Fatal("Can't find function")
	}

	return fset, f, fn, body
}

// Check performs type checking on the provided AST files.
// It creates and returns a fully type-checked *types.Package and *types.Info.
// Use this helper when testing analyzer components that require type information
// (e.g. for method lookup, type identity, or scope analysis).
func Check(tb testing.TB, fset *token.FileSet, f *ast.File) (*types.Package, *types.Info) {
	tb.Helper()

	info := &types.Info{
		Types:  make(map[ast.Expr]types.TypeAndValue),
		Defs:   make(map[*ast.Ident]types.Object),
		Uses:   make(map[*ast.Ident]types.Object),
		Scopes: make(map[ast.Node]*types.Scope),
	}

	conf := types.Config{Importer: importer.Default()}

	pkg, err := conf.Check(testpkg, fset, []*ast.File{f}, info)
	if err != nil {
		tb.Fatalf("failed to type Check source: %v", err)
	}

	return pkg, info
}

func wrapSource(src string) *bytes.Buffer {
	const (
		header     = "package " + testpkg + "\n\nfunc _() {\n"
		suffix     = "\n}"
		wrapperLen = len(header) + len(suffix)
	)

	var srcFile bytes.Buffer
	srcFile.Grow(wrapperLen + len(src))

	srcFile.WriteString(header) // ignore error
	srcFile.WriteString(src)    // ignore error
	srcFile.WriteString(suffix) // ignore error

	return &srcFile
}

func firstFuncDecl(f *ast.File) (fn *ast.FuncDecl, body inspector.Cursor) {
	root := inspector.New([]*ast.File{f}).Root()
	for c := range root.Preorder((*ast.FuncDecl)(nil)) {
		fn, body = c.Node().(*ast.FuncDecl), c.ChildAt(edge.FuncDecl_Body, -1)

		return fn, body
	}

	return nil, root
}
