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
	"go/types"
)

// Shadowing looks for a shadowed variable in parent scopes. It doesn't
// cross function scopes.
//
// Parameters:
//   - inner: The variable that may be shadowing another
//
// Returns:
//   - outer: The outer variable being shadowed (nil if none found)
func (s Index) Shadowing(inner *types.Var) (outer *types.Var) {
	scope := inner.Parent() // The scope the variable declaration lives in

	if _, ok := s[scope].(*ast.FuncType); ok {
		return nil // Variable declared at function top level - we don't cross them
	}

	name := inner.Name()

	// Search parent scopes for a variable with the same name and type
	for parent := scope.Parent(); parent != nil; parent = parent.Parent() {
		// Look for a variable with the same name in this scope
		shadowed := parent.Lookup(name)
		if shadowed == nil || shadowed.Pos() > inner.Pos() {
			if _, ok := s[parent].(*ast.FuncType); ok {
				break // Don't cross function boundaries
			}

			continue
		}

		outer, ok := shadowed.(*types.Var)
		if !ok || !types.Identical(outer.Type(), inner.Type()) {
			// Not a variable, or has different type (e.g., x := x.(T) type assertion)
			return nil
		}

		return outer // Found a shadowed variable with matching name and type
	}

	return nil
}
