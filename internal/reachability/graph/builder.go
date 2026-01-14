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

package graph

import (
	"fmt"
	"go/ast"
	"go/token"

	"fillmore-labs.com/scopeguard/internal/reachability/block"
	"fillmore-labs.com/scopeguard/internal/reachability/tracker"
)

// builder constructs the control flow graph.
// It traverses the AST and creates blocks and edges based on control flow semantics.
//
// The append* methods return the next basic [block] where statements should be added.
type builder struct {
	block.Factory                         // All blocks created during traversal
	labels        map[string]*LabelTarget // Maps label names to their target blocks
	targetScopes  branchTargetScopes      // Current break/continue/fallthrough targets
	forwardOnly   bool                    // Whether not to create backlinks

	tracker.Tracker
}

// appendStmtList appends a list of statements to the current block.
func (b *builder) appendStmtList(current *block.Block, list []ast.Stmt) *block.Block {
	for _, s := range list {
		current = b.appendStmt(current, s, nil)
	}

	return current
}

// appendStmt appends a single statement to the current block.
// labeled indicates if the statement has a label target (for break/continue/goto).
func (b *builder) appendStmt(current *block.Block, stmt ast.Stmt, labeled *LabelTarget) *block.Block {
	switch stmt := stmt.(type) {
	// keep-sorted start newline_separated=yes
	case *ast.AssignStmt:
		current.AddSimpleStmt(stmt)
		return current

	case *ast.BadStmt, *ast.DeferStmt, *ast.EmptyStmt, *ast.GoStmt, *ast.IncDecStmt, *ast.SendStmt:
		current.AddSimpleStmt(stmt)
		return current

	case *ast.BlockStmt:
		return b.appendStmtList(current, stmt.List)

	case *ast.BranchStmt:
		return b.appendBranchStmt(current, stmt)

	case *ast.DeclStmt:
		// Skip const and type declarations
		if d, ok := stmt.Decl.(*ast.GenDecl); ok && d.Tok == token.VAR {
			current.AddSimpleStmt(stmt)
		}

		return current

	case *ast.ExprStmt:
		current.AddSimpleStmt(stmt)

		if call, ok := stmt.X.(*ast.CallExpr); ok && b.CantReturn(call) {
			return b.New(stmt.End()) // unreachable after non-returning call
		}

		return current

	case *ast.ForStmt:
		return b.appendForStmt(current, stmt, labeled)

	case *ast.IfStmt:
		return b.appendIfStmt(current, stmt)

	case *ast.LabeledStmt:
		return b.appendLabeledStmt(current, stmt)

	case *ast.RangeStmt:
		return b.appendRangeStmt(current, stmt, labeled)

	case *ast.ReturnStmt:
		current.AddSimpleStmt(stmt)

		return b.New(stmt.End()) // unreachable after return

	case *ast.SelectStmt:
		return b.appendSelectStmt(current, stmt, labeled)

	case *ast.SwitchStmt:
		return b.appendSwitchStmt(current, stmt, labeled)

	case *ast.TypeSwitchStmt:
		return b.appendTypeSwitchStmt(current, stmt, labeled)

	default: // *ast.CaseClause and *ast.CommClause
		msg := fmt.Errorf("unexpected statement type: %T", stmt)
		panic(msg)
		// keep-sorted end
	}
}

// appendLabeledStmt handles labeled statements.
func (b *builder) appendLabeledStmt(current *block.Block, stmt *ast.LabeledStmt) *block.Block {
	labeled := b.labelTarget(stmt.Label)
	body := labeled.Body()
	body.SetStart(stmt.Stmt.Pos())

	current.Link(body)

	return b.appendStmt(body, stmt.Stmt, labeled)
}

// appendBranchStmt handles break, continue, goto, and fallthrough equivalents.
func (b *builder) appendBranchStmt(current *block.Block, stmt *ast.BranchStmt) *block.Block {
	var target *block.Block
	if stmt.Label == nil {
		target = b.targetScopes.branchTarget(stmt.Tok)
	} else {
		labeled := b.labelTarget(stmt.Label)
		target = labeled.BranchTarget(stmt.Tok)
	}

	current.AddSimpleStmt(stmt) // make current non-empty

	if target != nil {
		if b.forwardOnly && stmt.Tok == token.GOTO && target.Pos.IsValid() {
			// Existing label, which means backwards jump
			return b.New(stmt.End()) // unreachable after goto
		}

		current.Link(target)
	}

	return b.New(stmt.End()) // unreachable after break, continue, goto, or fallthrough
}

// labelTarget retrieves or creates a target for the given label.
func (b *builder) labelTarget(label *ast.Ident) *LabelTarget {
	if target, ok := b.labels[label.Name]; ok {
		return target
	}

	body := b.New(token.NoPos) // forward goto reference
	target := NewLabelTarget(body)
	b.labels[label.Name] = target

	return target
}

// appendIfStmt handles if statements.
func (b *builder) appendIfStmt(current *block.Block, stmt *ast.IfStmt) *block.Block {
	if stmt.Init != nil {
		current.AddSimpleStmt(stmt.Init)
	}

	current.AddExpr(stmt.Cond)

	after := b.New(stmt.End())     // after if
	body := b.New(stmt.Body.Pos()) // if body

	afterBody := b.appendStmtList(body, stmt.Body.List)
	afterBody.Link(after)

	elseBranch := after
	if stmt.Else != nil {
		elseBranch = b.New(stmt.Else.Pos()) // else branch

		afterElse := b.appendStmt(elseBranch, stmt.Else, nil)
		afterElse.Link(after)
	}

	current.LinkBranch(body, elseBranch)

	return after
}

// appendSwitchStmt handles expression switch statements.
func (b *builder) appendSwitchStmt(current *block.Block, stmt *ast.SwitchStmt, labeled *LabelTarget) *block.Block {
	if stmt.Init != nil {
		current.AddSimpleStmt(stmt.Init)
	}

	if stmt.Tag != nil {
		current.AddExpr(stmt.Tag)
	}

	return b.appendSwitchBody(current, stmt.Body, labeled, false)
}

// appendTypeSwitchStmt handles expression switch statements.
func (b *builder) appendTypeSwitchStmt(current *block.Block, stmt *ast.TypeSwitchStmt, labeled *LabelTarget) *block.Block {
	if stmt.Init != nil {
		current.AddSimpleStmt(stmt.Init)
	}

	current.AddSimpleStmt(stmt.Assign)

	return b.appendSwitchBody(current, stmt.Body, labeled, true)
}

// appendSwitchBody handles a switch statements body.
func (b *builder) appendSwitchBody(current *block.Block, cases *ast.BlockStmt, labeled *LabelTarget, typeSwitch bool) *block.Block {
	numCases := len(cases.List)
	if numCases == 0 {
		return current
	}

	after, old := b.newAfterBlock(labeled, cases.End()) // after switch

	// no default, switch can fall through
	defaultTarget := after

	// previous case expressions, linked current -> expr1 -> expr2 -> default
	prevExpr := current

	var prevBody *block.Block // case body

	nextBody := b.New(token.NoPos) // first switch case

	// See https://go.dev/ref/spec#Switch_statements
	for i, clause := range cases.List {
		clause := clause.(*ast.CaseClause)

		if clause.List == nil {
			defaultTarget = nextBody // default case
		} else {
			caseExpr := b.New(clause.Pos() + 4 /* len("case") */) // case expressions
			if !typeSwitch {
				caseExpr.AddExprs(clause.List)
			}

			// link previous case expressions to previous body and current case expressions, skip default case
			prevExpr.LinkClause(prevBody, caseExpr)
			prevBody, prevExpr = nextBody, caseExpr
		}

		// current case body
		body := nextBody
		body.SetStart(clause.Colon + 1)

		nextBody = nil
		if i < numCases-1 {
			nextBody = b.New(token.NoPos) // next switch case
		}

		fallthroughTarget := nextBody
		if typeSwitch {
			fallthroughTarget = nil
		}

		// While there can only be one fallthrough target, switches could be nested
		oldf := b.targetScopes.pushFallthrough(fallthroughTarget)

		body = b.appendStmtList(body, clause.Body)
		body.Link(after)

		b.targetScopes.popFallthrough(oldf)
	}

	// default case after all expressions
	prevExpr.LinkClause(prevBody, defaultTarget)

	b.popAfterBreak(old)

	return after
}

// appendSelectStmt handles select statements.
func (b *builder) appendSelectStmt(current *block.Block, stmt *ast.SelectStmt, labeled *LabelTarget) *block.Block {
	after, old := b.newAfterBlock(labeled, stmt.End()) // after select

	// First all the channel operands are evaluated
	operands := current

	// See https://go.dev/ref/spec#Select_statements
	for _, clause := range stmt.Body.List {
		switch clause := clause.(*ast.CommClause); stmt := clause.Comm.(type) {
		case nil: // default
			// No operand to evaluate

		case *ast.SendStmt: // ch <- value
			channelOperand := b.New(stmt.Pos()) // channel operand
			operands.Link(channelOperand)
			channelOperand.AddSimpleStmt(stmt)
			operands = channelOperand

		case *ast.AssignStmt: // x := <- ch
			channelOperand := b.New(token.Pos(int(stmt.TokPos) + len(stmt.Tok.String())) /* stmt.Rhs */) // channel operand
			operands.Link(channelOperand)
			channelOperand.AddExprs(stmt.Rhs)
			operands = channelOperand

		case *ast.ExprStmt: // <- ch
			channelOperand := b.New(stmt.Pos()) // channel operand
			operands.Link(channelOperand)
			channelOperand.AddExpr(stmt.X)
			operands = channelOperand

		default:
			msg := fmt.Errorf("unexpected communication clause: %T", stmt)
			panic(msg)
		}
	}

	prevDispatch := operands

	var prevBody *block.Block

	// Then, a random clause is selected
	for _, clause := range stmt.Body.List {
		clause := clause.(*ast.CommClause)

		dispatch := b.New(token.NoPos) // dispatch block
		prevDispatch.LinkClause(prevBody, dispatch)

		hasBody := len(clause.Body) > 0

		body := after
		if hasBody {
			body = b.New(clause.Colon + 1) // case body
		}

		prevDispatch, prevBody = dispatch, body

		if stmt, ok := clause.Comm.(*ast.AssignStmt); ok {
			assign := b.New(stmt.Pos()) // received values are assigned
			prevBody = assign

			assign.AddExprs(stmt.Lhs)
			assign.Link(body)
		}

		if hasBody {
			body = b.appendStmtList(body, clause.Body)
			body.Link(after)
		}
	}

	if prevBody != nil {
		prevDispatch.Link(prevBody)
	}

	b.popAfterBreak(old)

	return after
}

// appendForStmt handles for loops.
func (b *builder) appendForStmt(current *block.Block, stmt *ast.ForStmt, labeled *LabelTarget) *block.Block {
	if stmt.Init != nil {
		current.AddSimpleStmt(stmt.Init)
	}

	body := b.New(stmt.Body.Lbrace + 1)                // for body
	after, old := b.newAfterBlock(labeled, stmt.End()) // after for

	forever := stmt.Cond == nil

	cond := body
	if !forever {
		cond = b.New(stmt.Cond.Pos()) // for condition
		cond.AddExpr(stmt.Cond)
		cond.LinkBranch(body, after)
	}

	current.Link(cond)

	loopBack := cond
	if b.forwardOnly {
		loopBack = after
		if forever {
			loopBack = b.New(stmt.End()) // unreachable after endless loop
		}
	}

	post := loopBack
	if stmt.Post != nil {
		post = b.New(stmt.Post.Pos()) // for post statement
		post.AddSimpleStmt(stmt.Post)
		// The init statement may be a short variable declaration, but the post statement must not.
		// https://go.dev/ref/spec#For_clause

		post.Link(loopBack)
	}

	if labeled != nil {
		labeled.SetContinue(post)
	}

	oldc := b.targetScopes.pushContinue(post)

	bodyEnd := b.appendStmtList(body, stmt.Body.List)
	bodyEnd.Link(post)

	b.targetScopes.popContinue(oldc)
	b.popAfterBreak(old)

	return after
}

// appendRangeStmt handles range loops.
func (b *builder) appendRangeStmt(current *block.Block, stmt *ast.RangeStmt, labeled *LabelTarget) *block.Block {
	if stmt.Key != nil {
		current.AddExpr(stmt.Key)

		if stmt.Value != nil {
			current.AddExpr(stmt.Value)
		}
	}

	current.AddExpr(stmt.X)

	body := b.New(stmt.Body.Lbrace + 1)                // range body
	after, old := b.newAfterBlock(labeled, stmt.End()) // after range

	current.Link(body)

	continueTarget := body
	if b.forwardOnly {
		continueTarget = after
	}

	if labeled != nil {
		labeled.SetContinue(continueTarget)
	}

	oldc := b.targetScopes.pushContinue(continueTarget)

	if bodyEnd := b.appendStmtList(body, stmt.Body.List); b.forwardOnly {
		bodyEnd.Link(after)
	} else {
		bodyEnd.LinkBranch(body, after)
	}

	b.targetScopes.popContinue(oldc)
	b.popAfterBreak(old)

	return after
}

func (b *builder) newAfterBlock(labeled *LabelTarget, pos token.Pos) (after, old *block.Block) {
	after = b.New(pos) // after

	if labeled != nil {
		labeled.SetBreak(after)
	}

	old = b.targetScopes.pushBreak(after)

	return after, old
}

func (b *builder) popAfterBreak(old *block.Block) {
	b.targetScopes.popBreak(old)
}
