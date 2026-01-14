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
	"go/types"
	"reflect"
	"testing"

	. "fillmore-labs.com/scopeguard/internal/usage/check"
)

func TestNestedChecker_TrackAssignment(t *testing.T) {
	t.Parallel()

	const (
		OuterDecl token.Pos = 10
		InnerUse  token.Pos = 15
		OuterEnd  token.Pos = 20
		OtherUse  token.Pos = 25
		LaterUse  token.Pos = 30
	)

	v1 := types.NewVar(OuterDecl, nil, "v1", types.Typ[types.Int])
	id1decl := &ast.Ident{Name: "v1", NamePos: OuterDecl}
	id1use1 := &ast.Ident{Name: "v1", NamePos: InnerUse}
	id1use2 := &ast.Ident{Name: "v1", NamePos: LaterUse}

	v2 := types.NewVar(OuterEnd, nil, "v2", types.Typ[types.Int])
	id2use := &ast.Ident{Name: "v2", NamePos: OtherUse}

	tests := []struct {
		name     string
		enabled  bool
		ops      func(*NestedChecker)
		expected []NestedAssign
	}{
		{
			name:    "disabled",
			enabled: false,
			ops: func(nc *NestedChecker) {
				nc.TrackNestedAssignment(v1, id1decl, OuterEnd, 1) // Outer assignment
				nc.TrackNestedAssignment(v1, id1use1, InnerUse, 2) // Inner assignment
			},
			expected: nil,
		},
		{
			name:    "no_nesting",
			enabled: true,
			ops: func(nc *NestedChecker) {
				nc.TrackNestedAssignment(v1, id1decl, OuterEnd, 1)
				nc.TrackNestedAssignment(v1, id1use2, LaterUse, 2)
			},
			expected: nil,
		},
		{
			name:    "simple_nesting",
			enabled: true,
			ops: func(nc *NestedChecker) {
				nc.TrackNestedAssignment(v1, id1decl, OuterEnd, 1) // v1 assigned, ends at 20
				nc.TrackNestedAssignment(v1, id1use1, OuterEnd, 2) // v1 nested assign at 15
			},
			expected: []NestedAssign{{Ident: id1use1, Asgn: 1}},
		},
		{
			name:    "different_variables",
			enabled: true,
			ops: func(nc *NestedChecker) {
				nc.TrackNestedAssignment(v1, id1decl, OtherUse, 1)
				nc.TrackNestedAssignment(v2, id2use, OuterEnd, 2)
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nc := NewNestedChecker(tt.enabled)
			tt.ops(&nc)

			if got := nc.NestedAssigned(); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("NestedAssigned() = %v, want %v", got, tt.expected)
			}
		})
	}
}
