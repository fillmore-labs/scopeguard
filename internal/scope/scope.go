// Copyright 2025-2026 Oliver Eikemeier. All Rights Reserved.
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
	"go/token"
	"go/types"
	"iter"
)

// Index provides scope analysis for variable movement.
//
// It maps scopes to their corresponding AST nodes and provides methods to:
//   - Determine if a variable can safely be moved to a tighter scope
//   - Find the tightest "safe" scope that avoids breaking semantics
type Index map[*types.Scope]ast.Node

// NewIndex creates a scope analyzer from the type checker's scope map.
func NewIndex(info *types.Info) Index {
	s := make(Index, len(info.Scopes))
	for node, scope := range info.Scopes {
		s[scope] = node
	}

	return s
}

// Innermost finds the innermost scope containing a use, with special handling
// for case/select expressions.
//
// For most positions, this returns the innermost scope from the type checker. However,
// when a variable is used in a case or select expression (between "case" and ":" tokens),
// it adjusts the scope to the parent.
func (s Index) Innermost(declScope *types.Scope, pos token.Pos) *types.Scope {
	usageScope := declScope.Innermost(pos)
	switch usageScope {
	case declScope, nil:
		return usageScope
	}

	// Special handling: if the variable is used in case/select expression,
	// adjust scope to parent to prevent moving a declaration into the case body
	switch n := s[usageScope].(type) {
	case *ast.CaseClause:
		if pos < n.Colon {
			// The variable is used in the case expression: case x == 0:
			// Can't move x into this case's scope
			usageScope = usageScope.Parent()
		}

	case *ast.CommClause:
		if pos < n.Colon {
			// The variable is used in the send/receive: case ch <- x:
			// Can't move x into this select case's scope
			usageScope = usageScope.Parent()
		}
	}

	return usageScope
}

// ParentScope calculates parent, skips case scopes when the current scope is not in the body but the expression.
func (s Index) ParentScope(scope *types.Scope) *types.Scope {
	parent := scope.Parent()

	// Skip case scopes when the current scope is not in the body.
	// Note: The parent of *ast.CaseClause expressions is the switch expression
	if n, ok := s[parent].(*ast.CommClause); ok && scope.Pos() < n.Colon {
		parent = parent.Parent()
	}

	return parent
}

// ParentScopes yields a sequence of scopes from start up to (but not including) root.
func (s Index) ParentScopes(root, start *types.Scope) iter.Seq2[*types.Scope, struct{}] {
	return func(yield func(*types.Scope, struct{}) bool) {
		for scope := start; scope != root; scope = s.ParentScope(scope) {
			if scope == nil { // Reached the [types.Universe] scope
				panic("start scope is not in root")
			}

			if !yield(scope, struct{}{}) {
				break
			}
		}
	}
}
