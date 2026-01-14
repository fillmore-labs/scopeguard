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

// branchTargetScopes maintains the current branch targets representing nested
// control structures (loops, switches, selects).
type branchTargetScopes struct {
	currentBreak *block.Block

	currentContinue *block.Block

	currentFallthrough *block.Block
}

func (s *branchTargetScopes) branchTarget(tok token.Token) *block.Block {
	switch tok {
	case token.BREAK:
		return s.currentBreak

	case token.CONTINUE:
		return s.currentContinue

	case token.FALLTHROUGH:
		return s.currentFallthrough

	default:
		panic(fmt.Sprintf("unexpected branch token: %s", tok))
	}
}

// pushBreak sets the current "break" branch target scope, returning the old.
func (s *branchTargetScopes) pushBreak(b *block.Block) (old *block.Block) {
	old, s.currentBreak = s.currentBreak, b
	return old
}

// popBreak restores the previous "break" branch target scope.
func (s *branchTargetScopes) popBreak(old *block.Block) {
	s.currentBreak = old
}

// pushFallthrough sets the current "fallthrough" branch target scope, returning the old.
func (s *branchTargetScopes) pushFallthrough(b *block.Block) (old *block.Block) {
	old, s.currentFallthrough = s.currentFallthrough, b
	return old
}

// popFallthrough restores the previous "fallthrough" branch target scope.
func (s *branchTargetScopes) popFallthrough(old *block.Block) {
	s.currentFallthrough = old
}

// pushContinue sets the current "continue" branch target scope, returning the old.
func (s *branchTargetScopes) pushContinue(b *block.Block) (old *block.Block) {
	old, s.currentContinue = s.currentContinue, b
	return old
}

// popContinue restores the previous "continue" branch target scope.
func (s *branchTargetScopes) popContinue(old *block.Block) {
	s.currentContinue = old
}
