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

package typeutil_test

import (
	"go/ast"
	"go/types"
	"testing"

	. "fillmore-labs.com/scopeguard/internal/typeutil"
)

func TestFuncOf(t *testing.T) {
	t.Parallel()

	tests := [...]struct {
		name          string
		src           string
		wantIdentName string
		wantNonFunc   bool
	}{
		{
			name:          "simple function call",
			src:           `func myFunc() int { return 0 }; var _ = myFunc()`,
			wantIdentName: "myFunc",
		},
		{
			name:          "selector expression on package",
			src:           `import "strings"; var _ = strings.Clone("")`,
			wantIdentName: "Clone",
		},
		{
			name:          "method call on a variable",
			src:           `type S struct{}; func (s S) myMethod() int { return 0 }; var v S; var _ = v.myMethod()`,
			wantIdentName: "myMethod",
		},
		{
			name:          "method expression call",
			src:           `type S struct{}; func (s S) myMethod() int { return 0 }; var v S; var _ = (S).myMethod(v)`,
			wantIdentName: "myMethod",
		},
		{
			name:          "method field call",
			src:           `type S struct{ f func() int }; var v S; var _ = v.f()`,
			wantIdentName: "f",
			wantNonFunc:   true,
		},
		{
			name:          "generic function call with one type parameter",
			src:           `func myFunc[T any]() T { return *new(T) }; var _ = myFunc[int]()`,
			wantIdentName: "myFunc",
		},
		{
			name:          "generic function call with multiple type parameters",
			src:           `func myFunc[T, U any]() T { return *new(T) }; var _ = myFunc[int, string]()`,
			wantIdentName: "myFunc",
		},
		{
			name:          "parenthesized function call",
			src:           `func myFunc() int { return 0 }; var _ = (myFunc)()`,
			wantIdentName: "myFunc",
		},
		{
			name:          "call on function variable",
			src:           `var myFunc func() int; var _ = myFunc()`,
			wantIdentName: "myFunc",
			wantNonFunc:   true,
		},
		{
			name: "call on a function pointer",
			src:  `var myFunc *func() int; var _ = (*myFunc)()`,
		},
		{
			name:          "type conversion",
			src:           `type myFuncType func() int; var f myFuncType; var _ = myFuncType(f)`,
			wantIdentName: "myFuncType",
			wantNonFunc:   true,
		},
		{
			name:          "external type conversion",
			src:           `import "go/doc"; var _ = doc.Filter(nil)`,
			wantIdentName: "Filter",
			wantNonFunc:   true,
		},
		{
			name: "call on a function result",
			src:  `func myFunc() func() int { return nil }; var _ = (myFunc)()()`,
		},
		{
			name: "IndexExpr with non-type index",
			src:  `var a [1]func() int; var _ = a[0]()`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset, f := parseSource(t, tt.src)
			_, info := checkSource(t, fset, []*ast.File{f})
			call := lastDeclCallExpr(f)

			obj := FuncOf(info, call)

			want, got := tt.wantIdentName != "", obj != nil
			if want != got {
				t.Errorf("FuncOf()==nil want = %t, got %t", want, got)
			}

			if !want {
				return
			}

			if idName := obj.Name(); idName != tt.wantIdentName {
				t.Errorf("FuncOf().Name() = %q, want %q", idName, tt.wantIdentName)
			}

			if _, ok := obj.(*types.Func); !ok != tt.wantNonFunc {
				t.Errorf("FuncOf() non func = %t, want %t", !ok, tt.wantNonFunc)
			}
		})
	}
}

func lastDeclCallExpr(f *ast.File) *ast.CallExpr {
	lastDecl := f.Decls[len(f.Decls)-1]
	genDecl := lastDecl.(*ast.GenDecl)
	valSpec := genDecl.Specs[0].(*ast.ValueSpec)
	callExpr := valSpec.Values[0].(*ast.CallExpr)

	return callExpr
}
