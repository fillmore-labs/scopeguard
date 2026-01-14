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
	"go/types"
	"maps"
)

// UsageScope determines the usage scope of declared variables.
// It extends ScopeIndex with usage-specific scope analysis.
type UsageScope struct {
	Index
}

// NewUsageScope creates a new [UsageScope] instance.
func NewUsageScope(scopes Index) UsageScope {
	return UsageScope{Index: scopes}
}

// CommonAncestor finds the lowest common ancestor (LCA) of two scopes in the scope tree.
//
//   - declScope: The declaration scope (root of the subtree we're searching)
//   - currentScope: First scope (the current minimum scope)
//   - usageScope: Second scope (scope of the new use we're processing)
func (s UsageScope) CommonAncestor(declScope, currentScope, usageScope *types.Scope) *types.Scope {
	switch usageScope {
	case currentScope, // Same scope as before: no change needed
		declScope: // Tightest possible
		return usageScope
	}

	// Phase 1: Build a path from currentScope to declScope
	// This creates a set of all scopes in the path
	path := maps.Collect(s.ParentScopes(declScope, currentScope))

	// Phase 2: Walk from usageScope to declScope
	// Return the first scope that exists in both paths (the LCA)
	for scope := range s.ParentScopes(declScope, usageScope) {
		if _, ok := path[scope]; ok {
			return scope
		}
	}

	// If we reach here, LCA is the declScope itself
	// This means the two scopes are in completely different branches
	return declScope
}
