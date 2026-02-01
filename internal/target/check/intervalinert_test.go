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

package check_test

import (
	"go/ast"
	"go/token"
	"go/version"
	"runtime"
	"testing"

	. "fillmore-labs.com/scopeguard/internal/target/check"
	"fillmore-labs.com/scopeguard/internal/testsource"
)

func TestIntervalInert(t *testing.T) {
	t.Parallel()

	tests := [...]struct {
		name     string
		src      string
		interval func(*ast.BlockStmt) (start, end token.Pos)
		want     bool
		version  string
	}{
		// Empty intervals
		{
			name:     "empty_interval",
			src:      ``,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.Lbrace, b.Rbrace },
			want:     true,
		},
		{
			name:     "empty_interval_between_declarations",
			src:      `const c = 1; type T int`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.List[0].End(), b.List[1].Pos() },
			want:     true,
		},

		// Assignment statements
		{
			name:     "assignment_short_declaration",
			src:      `x := 1; _ = x`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.Lbrace, b.List[0].End() },
			want:     true,
		},
		{
			name:     "assignment_short_declaration_func",
			src:      `x := func() int { return 1 }; _ = x`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.Lbrace, b.List[0].End() },
			want:     false,
		},
		{
			name:     "assignment_reassignment",
			src:      `x := 1; x = 2; _ = x`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.List[0].End(), b.List[1].End() },
			want:     false,
		},

		// Var declarations
		{
			name:     "var_without_value",
			src:      `var x int; _ = x`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.Lbrace, b.List[0].End() },
			want:     true,
		},
		{
			name:     "var_with_runtime_value",
			src:      `var y = make([]int, 10); _ = y`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.Lbrace, b.List[0].End() },
			want:     true,
		},
		{
			name:     "var_with_literal",
			src:      `var z = 42; _ = z`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.Lbrace, b.List[0].End() },
			want:     true,
		},
		{
			name:     "var_with_const_expr",
			src:      `var z = "x" + "y"; _ = z`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.Lbrace, b.List[0].End() },
			want:     true,
		},
		{
			name:     "var_with_complex_const",
			src:      `var y = 1 + int(1); _ = y`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.Lbrace, b.List[0].End() },
			want:     true,
		},
		{
			name:     "var_with_new",
			src:      `var y = new(int); _ = y`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.Lbrace, b.List[0].End() },
			want:     true,
		},

		// Control flow statements
		{
			name:     "if_statement",
			src:      `x := 1; if a := x; a > 0 { _ = a }; for i := 0; i < 10; i++ { _ = i }`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.List[0].End(), b.List[1].End() },
			want:     false,
		},
		{
			name:     "for_loop",
			src:      `x := 1; if a := x; a > 0 { _ = a }; for i := 0; i < 10; i++ { _ = i }`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.List[1].End(), b.List[2].End() },
			want:     false,
		},

		// Concurrency statements
		{
			name:     "defer_statement",
			src:      `defer func() {}(); go func() {}()`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.Lbrace, b.List[0].End() },
			want:     false,
		},
		{
			name:     "go_statement",
			src:      `defer func() {}(); go func() {}()`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.List[0].End(), b.List[1].End() },
			want:     false,
		},

		// Mixed statements
		{
			name:     "mixed_with_side_effects",
			src:      `const c = 1; type T int; x := func() int { return 1 }; var y int; println(x, y)`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.List[1].End(), b.List[3].End() },
			want:     false,
		},
		{
			name:     "mixed_only_declarations",
			src:      `const c = 1; type T int; x := 1; var y int; println(x, y)`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.List[0].End(), b.List[3].End() },
			want:     true,
		},
		{
			name:     "new_with_expression",
			src:      `var y = new("&[]int{}"); _ = y`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.Lbrace, b.List[0].End() },
			want:     true,
			version:  "go1.26",
		},
		{
			name:     "new_with_complex_expression",
			src:      `var y = new(&[]int{}); _ = y`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.Lbrace, b.List[0].End() },
			want:     true,
			version:  "go1.26",
		},
		{
			name:     "new_assign_with_expression",
			src:      `y := new("&[]int{}"); _ = y`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.Lbrace, b.List[0].End() },
			want:     true,
			version:  "go1.26",
		},
		{
			name:     "new_assign_with_complex_expression",
			src:      `y := new(&[]int{}); _ = y`,
			interval: func(b *ast.BlockStmt) (start, end token.Pos) { return b.Lbrace, b.List[0].End() },
			want:     true,
			version:  "go1.26",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.version != "" && version.Compare(runtime.Version(), tt.version) < 0 {
				t.Skipf("needs minimal version %s", tt.version)
			}

			fset, f, fun, body := testsource.Parse(t, tt.src)
			_, info := testsource.Check(t, fset, f)

			start, end := tt.interval(fun.Body)

			if got, want := IntervalInert(info, body, nil, start, end), tt.want; got != want {
				t.Errorf("Got inert %t, expected %t", got, want)
			}
		})
	}
}
