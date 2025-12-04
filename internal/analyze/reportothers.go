// Copyright 2025 Oliver Eikemeier. All Rights Reserved.
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

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
)

// Report emits diagnostics for nested assigned of variables.
func (n NestedAssigned) Report(ctx context.Context, p *analysis.Pass, currentFile CurrentFile) {
	defer trace.StartRegion(ctx, "ReportNestedAssigned").End()

	for _, assignment := range n {
		if currentFile.HasNoLintComment(assignment.id.Pos()) {
			continue
		}

		p.Report(analysis.Diagnostic{
			Pos:     assignment.id.Pos(),
			End:     assignment.id.End(),
			Message: fmt.Sprintf("Nested reassignment of variable '%s' (sg:nst)", assignment.id.Name),
			Related: []analysis.RelatedInformation{{
				Pos:     assignment.stmt.Pos(),
				End:     assignment.stmt.End(),
				Message: "Inside this assign statement",
			}},
		})
	}
}

// Report emits diagnostics for variables used after previously shadowed.
func (s ShadowUsed) Report(ctx context.Context, p *analysis.Pass, in *inspector.Inspector, currentFile CurrentFile) {
	defer trace.StartRegion(ctx, "ReportShadowed").End()

	for _, shadowed := range s {
		use := in.At(shadowed.Use).Node()
		if currentFile.HasNoLintComment(use.Pos()) {
			continue
		}

		name, decl := shadowed.Var.Name(), in.At(shadowed.Decl).Node()
		p.Report(analysis.Diagnostic{
			Pos:     use.Pos(),
			End:     use.End(),
			Message: fmt.Sprintf("Identifier '%s' used after previously shadowed (sg:uas)", name),
			Related: []analysis.RelatedInformation{{Pos: decl.Pos(), End: decl.Pos(), Message: "After this declaration"}},
		})
	}
}
