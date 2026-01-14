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

package block

import (
	"go/ast"
	"go/token"
)

// Block represents a [basic Block] in the [control-flow graph].
// It is a sequence of expressions and statements with a single entry and exit point.
// It tracks its position in the source code and its successor blocks.
//
// [basic Block]: https://en.wikipedia.org/wiki/Basic_block
// [control-flow graph]: https://en.wikipedia.org/wiki/Control-flow_graph
type Block struct {
	Pos, End token.Pos // The beginning and end of the source range

	// The successors.
	//
	// For unconditional jumps, Successor1 is the only successor.
	// For conditional branches, Successor1 is the "then" branch,
	// Successor2 the "else" branch.
	Successor1, Successor2 *Block
}

// GetSourceRange returns the source code range of the block.
func (b *Block) GetSourceRange(fset *token.FileSet) (from, to token.Position) {
	from = fset.PositionFor(b.Pos, false)
	to = fset.PositionFor(b.End, false)

	return from, to
}

func (b *Block) isEmpty() bool {
	return !b.End.IsValid()
}

func (b *Block) cmp(a *Block) int {
	return int(b.Pos - a.Pos)
}

// AddExpr appends an expression to the block, updating its source code range to include the expression's range.
func (b *Block) AddExpr(expr ast.Expr) {
	b.update(expr.Pos(), expr.End())
}

// AddSimpleStmt adds a single statement to the block and updates its source code range to include the statement's range.
func (b *Block) AddSimpleStmt(stmt ast.Stmt) {
	b.update(stmt.Pos(), stmt.End())
}

// AddExprs appends a list of expressions to the block, updating its source code range to include all expressions.
func (b *Block) AddExprs(exprs []ast.Expr) {
	l := len(exprs)
	if l == 0 {
		return
	}

	pos, end := exprs[0].Pos(), exprs[l-1].End()

	b.update(pos, end)
}

// AddFields updates the block's source range to include the range of the provided [ast.FieldList].
func (b *Block) AddFields(fields *ast.FieldList) {
	if fields == nil {
		return
	}

	l := len(fields.List)
	if l == 0 {
		return
	}

	first := fields.List[0].Names
	if len(first) == 0 {
		return
	}

	last := fields.List[l-1].Names
	pos, end := first[0].Pos(), last[len(last)-1].End()

	b.update(pos, end)
}

// LinkClause sets the successors for a clause in a chain (switch/select).
//
// It links the current clause to the next clause in the chain, while optionally
// branching to a body if the clause is not the start of the chain.
//
//	current -> clause -> clause -> ...
//	              |         |
//	              v         v
//	            body      body
func (b *Block) LinkClause(body, next *Block) {
	if body == nil {
		b.Successor1 = next
		return
	}

	b.Successor1, b.Successor2 = body, next
}
