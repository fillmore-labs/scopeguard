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
	"go/token"

	"fillmore-labs.com/scopeguard/internal/reachability/block"
)

// LabelTarget represents the control flow targets for a labeled statement.
// A label can be the target of break, continue, or goto statements.
type LabelTarget struct {
	statement      *block.Block // The labeled statement itself
	breakTarget    *block.Block // Where to jump on 'break label'
	continueTarget *block.Block // Where to jump on 'continue label'
}

// NewLabelTarget creates a new label target with the given body source range.
// The break and continue targets are set later based on the statement type.
func NewLabelTarget(body *block.Block) *LabelTarget {
	return &LabelTarget{statement: body}
}

// Body returns the source range of the labeled statement itself.
func (l *LabelTarget) Body() *block.Block {
	return l.statement
}

// SetBreak sets the break target block for the labeled statement.
func (l *LabelTarget) SetBreak(b *block.Block) {
	l.breakTarget = b
}

// SetContinue sets the continue target block for the labeled statement.
func (l *LabelTarget) SetContinue(c *block.Block) {
	l.continueTarget = c
}

// BranchTarget returns the block that a branch statement should
// jump to based on the branch token type (BREAK, CONTINUE, or GOTO).
func (l *LabelTarget) BranchTarget(tok token.Token) *block.Block {
	switch tok {
	case token.BREAK:
		return l.breakTarget

	case token.CONTINUE:
		return l.continueTarget

	case token.GOTO:
		return l.statement

	default:
		panic(fmt.Sprintf("unexpected labeled branch token: %s", tok))
	}
}
