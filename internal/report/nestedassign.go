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

package report

import (
	"context"
	"fmt"
	"runtime/trace"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"

	"fillmore-labs.com/scopeguard/internal/astutil"
	"fillmore-labs.com/scopeguard/internal/usage"
)

// reportNestedAssigned emits diagnostics for nested assigns of variables.
func reportNestedAssigned(ctx context.Context, p *analysis.Pass, in *inspector.Inspector, currentFile astutil.CurrentFile, nested []usage.NestedAssign) {
	if len(nested) == 0 {
		return
	}

	defer trace.StartRegion(ctx, "ReportNestedAssigned").End()

	for _, assignment := range nested {
		if currentFile.NoLintComment(assignment.Ident.Pos()) {
			continue
		}

		stmt := assignment.Asgn.Node(in)

		p.Report(analysis.Diagnostic{
			Pos:     assignment.Ident.Pos(),
			End:     assignment.Ident.End(),
			Message: fmt.Sprintf("Nested reassignment of variable '%s' (sg:nst)", assignment.Ident.Name),
			Related: []analysis.RelatedInformation{{
				Pos:     stmt.Pos(),
				End:     stmt.End(),
				Message: "Inside this assign statement",
			}},
		})
	}
}
