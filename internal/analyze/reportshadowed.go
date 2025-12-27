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

package analyze

import (
	"context"
	"fmt"
	"runtime/trace"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
)

// Report emits diagnostics for nested assigns of variables.
func (n NestedAssigned) Report(ctx context.Context, p *analysis.Pass, in *inspector.Inspector, currentFile CurrentFile) {
	defer trace.StartRegion(ctx, "ReportNestedAssigned").End()

	for _, assignment := range n {
		if currentFile.NoLintComment(assignment.id.NamePos) {
			continue
		}

		stmt := in.At(assignment.asgn).Node()

		p.Report(analysis.Diagnostic{
			Pos:     assignment.id.Pos(),
			End:     assignment.id.End(),
			Message: fmt.Sprintf("Nested reassignment of variable '%s' (sg:nst)", assignment.id.Name),
			Related: []analysis.RelatedInformation{{
				Pos:     stmt.Pos(),
				End:     stmt.End(),
				Message: "Inside this assign statement",
			}},
		})
	}
}

// Report emits diagnostics for variables used after previously shadowed.
func (u UsedAfterShadow) Report(ctx context.Context, p *analysis.Pass, fdecl inspector.Cursor, currentFile CurrentFile, rename bool) bool {
	defer trace.StartRegion(ctx, "ReportShadowed").End()

	slices.SortFunc(u, func(a, b ShadowUse) int { return int(a.Use - b.Use) })

	var renamer *Renamer
	if rename {
		renamer = NewRenamer()
	}

	hadFixes := false

	in := fdecl.Inspector()

	for _, shadowed := range u {
		use := in.At(shadowed.Use).Node()
		if currentFile.NoLintComment(use.Pos()) {
			continue
		}

		suggestedFixes := renamer.Renames(p.TypesInfo, fdecl, shadowed.Var)

		if len(suggestedFixes) > 0 {
			hadFixes = true
		}

		name, decl := shadowed.Var.Name(), in.At(shadowed.Decl).Node()
		p.Report(analysis.Diagnostic{
			Pos:            use.Pos(),
			End:            use.End(),
			Message:        fmt.Sprintf("Identifier '%s' used after previously shadowed (sg:uas)", name),
			Related:        []analysis.RelatedInformation{{Pos: decl.Pos(), End: decl.Pos(), Message: "After this declaration"}},
			SuggestedFixes: suggestedFixes,
		})
	}

	return hadFixes
}
