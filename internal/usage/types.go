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

// DeclarationNode tracks a (re-)declaration node and its usages.
type DeclarationNode struct {
	Decl  astutil.NodeIndex
	Usage Flags
}

// Flags indicate how a variable is used.
//
//go:generate go tool bitmask -type Flags -output=types_bitmask.go
type Flags uint8

const (
	// UsageUsed indicates the variable declaration is used.
	UsageUsed Flags = 1 << iota // Used

	// UsageTypeChange indicates the variable redeclaration implies a type change.
	UsageTypeChange // TypeChange

	// UsageUntypedNil indicates the variable redeclaration is assigned to untyped nil.
	UsageUntypedNil // UntypedNil

	// UsageNone indicates the variable redeclaration is unused.
	UsageNone Flags = 0 // Unused

	// UsageUsedAndTypeChange represents a combination of [UsageUsed] and [UsageTypeChange] flags.
	UsageUsedAndTypeChange = UsageUsed | UsageTypeChange
)

// Result contains the scope analysis for all variable declarations from stage 1.
type Result struct {
	// Map from declaration indices to their computed scope ranges.
	scopeRanges map[astutil.NodeIndex]ScopeRange

	// Map of variables to declaration.
	declarations map[*types.Var][]DeclarationNode
}

// HasScopeRanges checks if any scope ranges are present in the result.
func (u Result) HasScopeRanges() bool {
	return len(u.scopeRanges) > 0
}

// AllScopeRanges returns all scope ranges in the result.
func (u Result) AllScopeRanges() iter.Seq2[astutil.NodeIndex, ScopeRange] {
	return maps.All(u.scopeRanges)
}

// AllDeclarations returns an iterator over all variables and their corresponding usage lists.
func (u Result) AllDeclarations() iter.Seq2[*types.Var, []DeclarationNode] {
	return maps.All(u.declarations)
}

// Diagnostics contains findings from the usage analysis stage.
type Diagnostics struct {
	Shadows []check.ShadowUse
	Nested  []check.NestedAssign
}
