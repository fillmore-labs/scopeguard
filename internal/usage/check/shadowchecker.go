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

	"fillmore-labs.com/scopeguard/internal/astutil"
	"fillmore-labs.com/scopeguard/internal/scope"
)

// ShadowChecker tracks variable shadowing and usage of variables while they are shadowed.
//
// It is designed to be embedded in other analyzers (like usageCollector) to add shadow detection capabilities.
type ShadowChecker struct {
	// shadowed maps shadowed variables.
	shadowed map[*types.Var]shadowInfo

	// usedAfterShadow collects usage of variables used after previously shadowed.
	usedAfterShadow []ShadowUse
}

// NewShadowChecker creates a new ShadowChecker instance.
//
// If enabled is false, shadow tracking is disabled and the checker is a no-op that uses minimal memory.
func NewShadowChecker(enabled bool) ShadowChecker {
	var sc ShadowChecker

	if enabled {
		sc.shadowed = make(map[*types.Var]shadowInfo)
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

	// decl is the inspector index of the inner declaration that shadows the outer variable.
	decl astutil.NodeIndex
}

// shadowing reports whether the given position falls within the active shadowing window.
// A position is shadowed if it's after the start and before the end (if set).
func (s shadowInfo) shadowing(pos token.Pos) bool {
	return pos >= s.start && (!s.end.IsValid() || pos < s.end) && s.ignore != pos
}

// RecordShadowingDeclaration checks if the variable v shadows another in parent scopes and records it.
func (sc *ShadowChecker) RecordShadowingDeclaration(scopes scope.UsageScope, v *types.Var, id *ast.Ident, decl astutil.NodeIndex) {
	if sc.shadowed == nil {
		return
	}

	if s, start := scopes.Shadowing(v, id.NamePos); s != nil {
		sc.shadowed[s] = shadowInfo{start: start, end: token.NoPos, decl: decl}
	}
}

// RecordShadowedUse checks if the variable v is shadowed at the given position.
// If it is, it records the usage.
func (sc *ShadowChecker) RecordShadowedUse(v *types.Var, pos token.Pos, idx astutil.NodeIndex) {
	if s, ok := sc.shadowed[v]; ok && s.shadowing(pos) {
		sc.recordUsedAfterShadow(v, idx, s.decl)
	}
}

// recordUsedAfterShadow tracks the usage of a variable after it has been previously shadowed.
func (sc *ShadowChecker) recordUsedAfterShadow(v *types.Var, use, decl astutil.NodeIndex) {
	sc.usedAfterShadow = append(sc.usedAfterShadow, ShadowUse{Var: v, Use: use, Decl: decl})

	delete(sc.shadowed, v) // record only the first usage
}

// RecordAssignment updates the shadowing information for a variable when it is reassigned.
// It marks the end of the shadowing range or removes the variable from the shadowed map.
//
// Called when an outer variable that was previously shadowed is reassigned.
// This "clears" the shadow, assuming the assignment indicates intentional use of the outer variable.
//
// Note: This heuristic is lexically based, not control-flow sensitive.
// An assignment inside an if/switch block clears the shadow for subsequent lines.
func (sc *ShadowChecker) RecordAssignment(v *types.Var, id *ast.Ident, assignmentDone token.Pos) {
	s, ok := sc.shadowed[v]
	if !ok {
		return
	}

	if !s.end.IsValid() {
		// First reassignment: set the shadow end position
		s.ignore = id.NamePos
		s.end = assignmentDone
		sc.shadowed[v] = s
	} else {
		// Already has an end position: shadow is fully resolved, remove from tracking
		delete(sc.shadowed, v)
	}
}

// UpdateShadows updates shadow tracking when variables are assigned.
// When a shadowed outer variable is reassigned, the shadow "ends" at that point,
// as the outer variable has a new value.
//
// Note: This is lexically based, not control-flow sensitive. An assignment inside
// an `if` block or switch `case` clears the shadow for subsequent lines.
func (sc *ShadowChecker) UpdateShadows(v *types.Var, id *ast.Ident, assignmentDone token.Pos) {
	// Was the assigned variable shadowed?
	s, ok := sc.shadowed[v]
	if !ok {
		return
	}

	// Update the shadow end position based on the current state:
	switch hasEnd := s.end.IsValid(); {
	case !hasEnd:
		// No end is set: This is the first reassignment, mark shadow as ending after this assignment
		s.ignore = id.NamePos
		s.end = assignmentDone
		sc.shadowed[v] = s

	case id.NamePos >= s.end:
		// We've passed the end: Shadow is done, remove from tracking
		delete(sc.shadowed, v)

	default:
		// We're before the end: We're in a nested scope (e.g., function literal)
		if assignmentDone < s.end {
			// Update to the earlier assignment position
			s.ignore = id.NamePos
			s.end = assignmentDone
			sc.shadowed[v] = s
		}
	}
}
