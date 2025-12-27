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
	"go/ast"
	"go/token"
	"go/types"

	"fillmore-labs.com/scopeguard/internal/astutil"
)

// handleIdent processes identifier usages.
func (c *collector) handleIdent(id *ast.Ident, idx astutil.NodeIndex) {
	v, ok := c.TypesInfo.Uses[id].(*types.Var)
	if !ok {
		return
	}

	decl, ok := c.current[v]
	if !ok || decl.ignore == id.NamePos {
		return // ignore usage on LHS of AssignStmt
	}

	c.RecordShadowedUse(v, id.NamePos, idx)

	usage := c.attributeDeclaration(v, decl.start < id.NamePos)
	if usage == nil {
		return
	}

	usage.Usage |= UsageUsed

	c.updateUsageScope(usage.Decl, v, id)
}

// handleNamedResults marks named result parameters as used when a bare return is encountered.
func (c *collector) handleNamedResults(idx astutil.NodeIndex, results *ast.FieldList, pos token.Pos) {
	if results == nil {
		return
	}

	for _, names := range results.List {
		for _, id := range names.Names {
			if id.Name == "_" {
				continue // blank identifier
			}

			v, ok := c.TypesInfo.Defs[id].(*types.Var)
			if !ok {
				continue
			}

			c.RecordShadowedUse(v, pos, idx)

			usages := c.usages[v]
			if len(usages) == 0 {
				continue
			}

			usage := &usages[len(usages)-1]

			usage.Usage |= UsageUsed

			c.notMovable(usage.Decl, v)
		}
	}
}

// attributeDeclaration returns the declaration that a variable usage should be attributed to.
// current indicates whether the usage occurs within the scope of the current or previous declaration.
func (c *collector) attributeDeclaration(v *types.Var, current bool) *NodeUsage {
	usages := c.usages[v]
	switch usageCount := len(usages); {
	case current && usageCount > 0:
		// The usage is attributed to the most recent declaration.
		return &usages[usageCount-1]

	case !current && usageCount > 1:
		// When a variable appears on the RHS of its own reassignment (e.g., x := x + 1),
		// the usage belongs to the previous declaration, not the new one being defined.
		return &usages[usageCount-2] // Use penultimate declaration

	default:
		return nil // No previous declaration to attribute to
	}
}

// updateUsageScope updates the scope range for a variable usage.
func (c *collector) updateUsageScope(decl astutil.NodeIndex, v *types.Var, id *ast.Ident) {
	if c.scopeRanges == nil {
		return
	}

	declScope := v.Parent()
	currentRange, hasRange := c.scopeRanges[decl]

	if hasRange {
		if currentRange.Decl != declScope {
			astutil.InternalError(c.Pass, id, "Different declaration scopes recorded for '%s'", v.Name())
		}

		if currentRange.Usage == declScope {
			return // Already at the innermost scope (can't move tighter)
		}
	}

	// Find the innermost scope containing this use
	usageScope := c.Innermost(declScope, id.NamePos)

	if hasRange {
		// Compute the minimum scope that contains all uses so far
		usageScope = c.CommonAncestor(declScope, currentRange.Usage, usageScope)

		if usageScope == currentRange.Usage {
			return // Unchanged
		}
	}

	// Set the target scope
	c.scopeRanges[decl] = ScopeRange{Decl: declScope, Usage: usageScope}
}
