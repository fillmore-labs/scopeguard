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

package scope

import (
	"go/ast"

	"golang.org/x/tools/go/ast/inspector"
)

// SelectIndex maps each [ast.CommClause] to its parent [ast.SelectStmt].
type SelectIndex map[*ast.CommClause]*ast.SelectStmt

// NewSelectIndex records every [ast.SelectStmt] in the body.
func NewSelectIndex(body inspector.Cursor) SelectIndex {
	var s SelectIndex

	for sel := range body.Preorder((*ast.SelectStmt)(nil)) {
		if s == nil {
			s = make(SelectIndex)
		}

		stmt := sel.Node().(*ast.SelectStmt)
		s.AddSelectStmt(stmt)
	}

	return s
}

// AddSelectStmt processes a select statement, mapping its communication clauses to the statement itself for tracking.
func (s SelectIndex) AddSelectStmt(stmt *ast.SelectStmt) {
	for _, clause := range stmt.Body.List {
		if clause, ok := clause.(*ast.CommClause); ok {
			s[clause] = stmt
		}
	}
}

// Stmt returns the parent *[ast.SelectStmt] associated with the given *[ast.CommClause].
func (s SelectIndex) Stmt(clause *ast.CommClause) *ast.SelectStmt {
	return s[clause]
}
