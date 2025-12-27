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

package usage

import (
	"go/types"
	"iter"
	"maps"

	"fillmore-labs.com/scopeguard/internal/astutil"
	"fillmore-labs.com/scopeguard/internal/usage/check"
)

// ScopeRange represents the scope range for a declaration.
type ScopeRange struct {
	// Decl is the scope where the variable was declared
	Decl,
	// Usage is the tightest scope containing all uses
	Usage *types.Scope
}

// NodeUsage tracks a single usage of a declaration.
type NodeUsage struct {
	Decl  astutil.NodeIndex
	Usage Flags
}

// Flags indicates how a variable is used.
type Flags uint8

const (
	// UsageUsed indicates the variable declaration is used.
	UsageUsed Flags = 1 << iota

	// UsageTypeChange indicates the variable redeclaration implies a type change.
	UsageTypeChange

	// UsageUntypedNil indicates the variable redeclaration is assigned to untyped nil.
	UsageUntypedNil

	// UsageNone indicates the variable declaration is unused.
	UsageNone Flags = 0

	// UsageUsedAndTypeChange represents a combination of [UsageUsed] and [UsageTypeChange] flags.
	UsageUsedAndTypeChange = UsageUsed | UsageTypeChange
)

// Used indicates the variable declaration is used.
func (f Flags) Used() bool {
	return f&UsageUsed != 0
}

// TypeChange indicates the variable redeclaration implies a type change.
func (f Flags) TypeChange() bool {
	return f&UsageTypeChange != 0
}

// UntypedNil indicates the variable redeclaration is assigned to untyped nil.
func (f Flags) UntypedNil() bool {
	return f&UsageUntypedNil != 0
}

// UsedAndTypeChange represents a combination of [Flags.Used] and [Flags.TypeChange].
func (f Flags) UsedAndTypeChange() bool {
	return f&UsageUsedAndTypeChange == UsageUsedAndTypeChange
}

// Result contains the scope analysis for all variable declarations from stage 1.
type Result struct {
	// Map from declaration indices to their computed scope ranges.
	scopeRanges map[astutil.NodeIndex]ScopeRange

	// Map of variables to usage.
	usages map[*types.Var][]NodeUsage
}

// HasScopeRanges checks if any scope ranges are present in the result.
func (u Result) HasScopeRanges() bool {
	return len(u.scopeRanges) > 0
}

// AllScopeRanges returns all scope ranges in the result.
func (u Result) AllScopeRanges() iter.Seq2[astutil.NodeIndex, ScopeRange] {
	return maps.All(u.scopeRanges)
}

// AllUsages returns an iterator over all variables and their corresponding usage lists.
func (u Result) AllUsages() iter.Seq2[*types.Var, []NodeUsage] {
	return maps.All(u.usages)
}

// Diagnostics contains findings from the usage analysis stage.
type Diagnostics struct {
	Shadows []ShadowUse
	Nested  []NestedAssign
}

type (
	// ShadowUse contains information about a variable use after previously shadowed.
	ShadowUse = check.ShadowUse
	// NestedAssign contains information about a nested variable assign.
	NestedAssign = check.NestedAssign
)
