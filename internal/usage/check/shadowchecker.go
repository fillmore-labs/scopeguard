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

package check

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"golang.org/x/tools/go/ast/inspector"

	"fillmore-labs.com/scopeguard/internal/astutil"
	"fillmore-labs.com/scopeguard/internal/scope"
)

// ShadowChecker tracks variable shadowing and usage of variables while they are shadowed.
//
// It is designed to be embedded in other analyzers (like usageCollector) to add shadow detection capabilities.
type ShadowChecker struct {
	// shadowed maps shadowed variables.
	shadowed map[*types.Var]shadowInfo

	// selects maps communication clauses to its enclosing select.
	selects scope.SelectIndex

	// usedAfterShadow collects usage of variables used after previously shadowed.
	usedAfterShadow []ShadowUse
}

// NewShadowChecker creates a new ShadowChecker instance.
//
// If enabled is false, shadow tracking is disabled and the checker is a no-op that uses minimal memory.
func NewShadowChecker(body inspector.Cursor, enabled bool) ShadowChecker {
	var sc ShadowChecker

	if enabled {
		sc.shadowed = make(map[*types.Var]shadowInfo)
		sc.selects = scope.NewSelectIndex(body)
	}

	return sc
}

// UsedAfterShadow returns the list of variables that were used after being shadowed.
func (sc *ShadowChecker) UsedAfterShadow() []ShadowUse {
	slices.SortFunc(sc.usedAfterShadow, func(a, b ShadowUse) int { return int(a.Use - b.Use) })

	return sc.usedAfterShadow
}

// shadowInfo tracks when an outer variable is shadowed by an inner declaration.
type shadowInfo struct {
	// start is the position where shadowing begins (end of the shadowing declaration).
	// end is the position where shadowing ends (end of reassignment to outer variable, or NoPos if not yet reassigned).
	start, end token.Pos

	// ignore is the position of the identifier in the reassignment statement itself.
	// This prevents the reassignment from being flagged as a "use while shadowed".
	ignore token.Pos

	// id is the inner identifier declaration that shadows the outer variable.
	id *ast.Ident
}

// shadowing reports whether the given position falls within the shadowing window.
func (s shadowInfo) shadowing(pos token.Pos) bool {
	return pos >= s.start && (!s.end.IsValid() || pos < s.end) && s.ignore != pos
}

// CheckDeclarationShadowing checks if the variable shadows another in parent scopes and records it.
func (sc *ShadowChecker) CheckDeclarationShadowing(scopes scope.UsageScope, variable *types.Var, id *ast.Ident) {
	if sc.shadowed == nil {
		return
	}

	if shadowed, boundary := scopes.Shadowing(sc.selects, variable); shadowed != nil {
		sc.shadowed[shadowed] = shadowInfo{start: boundary, end: token.NoPos, id: id}
	}
}

// CheckUseAfterShadowed checks if the variable is used at the given position after shadowed.
// If it is, it records the usage.
func (sc *ShadowChecker) CheckUseAfterShadowed(variable *types.Var, namePos token.Pos, use astutil.NodeIndex) {
	if s, ok := sc.shadowed[variable]; ok && s.shadowing(namePos) {
		sc.usedAfterShadow = append(sc.usedAfterShadow, ShadowUse{Var: variable, Ident: s.id, Use: use})

		delete(sc.shadowed, variable) // record only the first usage
	}
}

// UpdateShadows updates shadow tracking when variables are assigned.
// When a shadowed outer variable is reassigned, the shadow "ends" at that point,
// as the outer variable has a new value.
//
// Note: This is lexically based, not control-flow sensitive.
func (sc *ShadowChecker) UpdateShadows(v *types.Var, namePos, assignmentDone token.Pos) {
	// Was the assigned variable shadowed?
	switch s, ok := sc.shadowed[v]; {
	case !ok:
		// Not shadowed.

	case namePos < s.start:
		// Assignment before we start complaining (e.g. in an else branch).

	case s.end.IsValid() && namePos >= s.end:
		// Assignment after shadow already ended.
		delete(sc.shadowed, v)

	case !s.end.IsValid() || assignmentDone < s.end:
		// First reassignment or nested reassignment.
		// We record the end of the shadow at this assignment.
		// If we already have an end position (from an outer scope assignment),
		// We update if this assignment finishes *earlier* (narrowing the shadow window).
		s.ignore = namePos
		s.end = assignmentDone
		sc.shadowed[v] = s
	}
}
