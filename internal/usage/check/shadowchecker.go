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
	"fillmore-labs.com/scopeguard/internal/reachability"
	"fillmore-labs.com/scopeguard/internal/scope"
)

// ShadowChecker tracks variable shadowing and usage of variables while they are shadowed.
//
// It is designed to be embedded in other analyzers (like usageCollector) to add shadow detection capabilities.
type ShadowChecker struct {
	// shadowed maps shadowed variables.
	shadowed map[*types.Var]shadowInfo

	// Control-flow graph to test reachability
	*reachability.Graph

	// usedAfterShadow collects usage of variables used after previously shadowed.
	usedAfterShadow []ShadowUse

	// firstUseOnly indicates whether only the first use of a variable after being shadowed should be recorded.
	firstUseOnly bool
}

// NewShadowChecker creates a new ShadowChecker instance.
//
// If enabled is false, shadow tracking is disabled and the checker is a no-op that uses minimal memory.
func NewShadowChecker(enabled, firstUseOnly bool) ShadowChecker {
	var sc ShadowChecker

	if enabled {
		sc.shadowed = make(map[*types.Var]shadowInfo)
		sc.firstUseOnly = firstUseOnly
	}

	return sc
}

// ShadowCheckerEnabled reports whether the shadow checker is enabled.
func (sc *ShadowChecker) ShadowCheckerEnabled() bool {
	return sc.shadowed != nil
}

// UsedAfterShadow returns the list of variables that were used after being shadowed.
func (sc *ShadowChecker) UsedAfterShadow() []ShadowUse {
	slices.SortFunc(sc.usedAfterShadow, func(a, b ShadowUse) int { return int(a.Use - b.Use) })

	return sc.usedAfterShadow
}

// shadowInfo tracks when an outer variable is shadowed by an inner declaration.
type shadowInfo struct {
	// end is the position where shadowing ends (end of reassignment to outer variable, or NoPos if not yet reassigned).
	end token.Pos

	// ignore is the position of the identifier in the reassignment statement itself.
	// This prevents the reassignment from being flagged as a "use while shadowed".
	ignore token.Pos

	// inner is the end of the shadowig scope.
	start token.Pos

	// shadowPos is the position of the inner identifier declaration that shadows the outer variable.
	shadowPos token.Pos

	// reassigns is a list of positions from where the variable is considered reassigned
	reassigns []token.Pos
}

// shadowing reports whether the given position falls within the shadowing window.
func (s shadowInfo) shadowing(pos token.Pos) bool {
	return pos >= s.start && (!s.end.IsValid() || pos < s.end) && s.ignore != pos
}

// CheckDeclarationShadowing checks if the variable shadows another in parent scopes and records it.
func (sc *ShadowChecker) CheckDeclarationShadowing(scopes scope.UsageScope, variable *types.Var, shadowPos token.Pos) {
	if sc.shadowed == nil {
		return
	}

	if outer := scopes.Shadowing(variable); outer != nil {
		sc.shadowed[outer] = shadowInfo{end: token.NoPos, start: variable.Parent().End(), shadowPos: shadowPos}
	}
}

// CheckUseAfterShadowed checks if the variable is used at the given position after shadowed.
// If it is, it records the usage.
func (sc *ShadowChecker) CheckUseAfterShadowed(variable *types.Var, namePos token.Pos, use astutil.NodeIndex) {
	s, ok := sc.shadowed[variable]
	if !ok || !s.shadowing(namePos) {
		return
	}

	// Is this usage reachable from the shadowing?
	if reachable, ok := sc.Reachable(s.shadowPos, namePos); ok && !reachable {
		return
	}

	// Do we have a reachable reassign in a subscope?
	for _, r := range s.reassigns {
		if reachable, ok := sc.Reachable(r-1, namePos); ok && reachable {
			return
		}
	}

	sc.usedAfterShadow = append(sc.usedAfterShadow, ShadowUse{Var: variable, ShadowPos: s.shadowPos, Use: use})

	// Report only the first use
	if sc.firstUseOnly {
		delete(sc.shadowed, variable)
	}
}

// UpdateShadows updates shadow tracking when variables are assigned at declaration scope.
// When a shadowed outer variable is reassigned, the shadow "ends" at that point,
// as the outer variable has a new value.
//
// Note: This is lexically based, not control-flow sensitive.
func (sc *ShadowChecker) UpdateShadows(v *types.Var, id *ast.Ident, assignmentDone token.Pos) {
	// Was the assigned variable shadowed?
	switch s, ok := sc.shadowed[v]; {
	case !ok:
		// Not shadowed.

	case s.end.IsValid() && id.NamePos >= s.end:
		// Assignment after shadow already ended.
		delete(sc.shadowed, v)

	case !s.end.IsValid() || assignmentDone < s.end:
		// Reassignment, we record the end of the shadow at this assignment.
		// If we already have an end position (from an outer scope assignment),
		// We update if this assignment finishes *earlier* (narrowing the shadow window).
		s.ignore = id.NamePos
		s.end = assignmentDone
		sc.shadowed[v] = s
	}
}

// UpdateShadowsWithReachability updates shadow tracking when variables are assigned in a subscope.
func (sc *ShadowChecker) UpdateShadowsWithReachability(v *types.Var, id *ast.Ident, assignmentDone token.Pos) {
	// Was the assigned variable shadowed?
	switch s, ok := sc.shadowed[v]; {
	case !ok:
		// Not shadowed.

	case s.end.IsValid() && id.NamePos >= s.end:
		// Assignment after shadow already ended.
		delete(sc.shadowed, v)

	case !s.end.IsValid() || assignmentDone < s.end:
		s.ignore = id.NamePos
		if reachable, ok := sc.Reachable(s.shadowPos, id.NamePos); !ok || reachable {
			// We record the reassignment when reachable.
			s.reassigns = append(s.reassigns, assignmentDone)
		}
		sc.shadowed[v] = s
	}
}
