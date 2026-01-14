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

package tracker_test

import (
	"go/token"
	"go/types"
	"testing"

	. "fillmore-labs.com/scopeguard/internal/reachability/tracker"
)

func TestFuncNameOf(t *testing.T) {
	t.Parallel()

	pkg := types.NewPackage("example.com/testpkg", "testpkg")

	typeName := types.NewTypeName(token.NoPos, pkg, "MyType", nil)
	emptystruct := types.NewStruct(nil, nil)
	named := types.NewNamed(typeName, emptystruct, nil)
	aliasName := types.NewTypeName(token.NoPos, pkg, "MyAlias", nil)
	alias := types.NewAlias(aliasName, types.NewPointer(named))

	tests := [...]struct {
		name         string
		fun          *types.Func
		wantFuncName string
	}{
		{
			name: "simple function call",
			fun: func() *types.Func {
				sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)

				return types.NewFunc(token.NoPos, pkg, "myFunc", sig)
			}(),
			wantFuncName: "example.com/testpkg.myFunc",
		},
		{
			name: "simple value method call",
			fun: func() *types.Func {
				recv := types.NewParam(token.NoPos, pkg, "", named)
				sig := types.NewSignatureType(recv, nil, nil, nil, nil, false)

				return types.NewFunc(token.NoPos, pkg, "myFunc", sig)
			}(),
			wantFuncName: "(example.com/testpkg.MyType).myFunc",
		},
		{
			name: "simple pointer method call",
			fun: func() *types.Func {
				recv := types.NewParam(token.NoPos, pkg, "", types.NewPointer(named))
				sig := types.NewSignatureType(recv, nil, nil, nil, nil, false)

				return types.NewFunc(token.NoPos, pkg, "myFunc", sig)
			}(),
			wantFuncName: "(example.com/testpkg.MyType).myFunc",
		},
		{
			name: "alias pointer method call",
			fun: func() *types.Func {
				recv := types.NewParam(token.NoPos, pkg, "", alias)
				sig := types.NewSignatureType(recv, nil, nil, nil, nil, false)

				return types.NewFunc(token.NoPos, pkg, "myFunc", sig)
			}(),
			wantFuncName: "(example.com/testpkg.MyType).myFunc",
		},
		{
			name: "interface method call",
			fun: func() *types.Func {
				sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
				iface := types.NewInterfaceType([]*types.Func{
					types.NewFunc(token.NoPos, pkg, "myFunc", sig),
				}, nil).Complete()

				return iface.Method(0)
			}(),
			wantFuncName: "(interface).myFunc",
		},
		{
			name: "function without package",
			fun: func() *types.Func {
				sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)

				return types.NewFunc(token.NoPos, nil, "myFunc", sig)
			}(),
			wantFuncName: "myFunc",
		},
		{
			name: "method on type without package",
			fun: func() *types.Func {
				return types.Universe.Lookup("error").Type().Underlying().(*types.Interface).Method(0)
			}(),
			wantFuncName: "(error).Error",
		},
		{
			name: "invalid method call",
			fun: func() *types.Func {
				recv := types.NewParam(token.NoPos, pkg, "", emptystruct)
				sig := types.NewSignatureType(recv, nil, nil, nil, nil, false)

				return types.NewFunc(token.NoPos, pkg, "myFunc", sig)
			}(),
			wantFuncName: "(<invalid>).myFunc",
		},
		{
			name: "invalid pointer method call",
			fun: func() *types.Func {
				recv := types.NewParam(token.NoPos, pkg, "", types.NewPointer(emptystruct))
				sig := types.NewSignatureType(recv, nil, nil, nil, nil, false)

				return types.NewFunc(token.NoPos, pkg, "myFunc", sig)
			}(),
			wantFuncName: "(<invalid>).myFunc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if name := FuncNameOf(tt.fun); name.String() != tt.wantFuncName {
				t.Errorf("FuncNameOf() = %q, want %q", name, tt.wantFuncName)
			}
		})
	}
}
