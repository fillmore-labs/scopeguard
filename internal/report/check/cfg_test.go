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
	"testing"

	. "fillmore-labs.com/scopeguard/internal/report/check"
	"fillmore-labs.com/scopeguard/internal/testsource"
)

func TestReachable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		src      string
		from, to ast.Node
		want     bool
	}{
		{
			name: "basic",
			src:  `var x int; if true { _ = x }`,
			from: (*ast.ValueSpec)(nil), to: (*ast.AssignStmt)(nil),
			want: true,
		},
		{
			name: "loop",
			src:  `for { var x int;  _ = x }`,
			from: (*ast.ValueSpec)(nil), to: (*ast.AssignStmt)(nil),
			want: true,
		},
		{
			name: "loopback",
			src:  `for { var x int;  _ = x }`,
			from: (*ast.AssignStmt)(nil), to: (*ast.ValueSpec)(nil),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset, f, _, body := testsource.Parse(t, tt.src)
			_, info := testsource.Check(t, fset, f)

			cfg := NewCFG(info, body.Node().(*ast.BlockStmt))

			var from, to token.Pos
			for n := range body.Preorder(tt.from) {
				from = n.Node().Pos()
				break
			}

			for n := range body.Preorder(tt.to) {
				to = n.Node().Pos()
				break
			}

			got, ok := cfg.Reachable(from, to)

			if !ok {
				t.Fatal("Range not found")
			}

			if got != tt.want {
				t.Errorf("Expected reachable %t, got %t", tt.want, got)
			}
		})
	}
}
