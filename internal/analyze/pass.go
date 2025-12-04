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
	"errors"
	"fmt"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// pass wraps *[analysis.Pass] with helper methods.
type pass struct {
	*analysis.Pass
}

// newPass creates a pass wrapper from an [*analysis.Pass].
func newPass(ap *analysis.Pass) pass {
	return pass{
		Pass: ap,
	}
}

// inspector retrieves the [inspector.Inspector] from the pass results.
//
// The inspector provides efficient AST traversal with cursor-based navigation,
// which is more ergonomic than raw AST traversal for our use case.
//
// Returns an error if the [inspect.Analyzer] result is missing.
func (p pass) inspector() (*inspector.Inspector, error) {
	in, ok := p.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("scopeguard: %s %w", inspect.Analyzer.Name, ErrResultMissing)
	}

	return in, nil
}

// ErrResultMissing is returned when a required analyzer result is missing.
// This typically indicates a configuration error where the analyzer's
// Requires field is not properly set.
var ErrResultMissing = errors.New("analyzer result missing")

// reportInternalError reports an internal error diagnostic.
// These errors indicate bugs in the analyzer logic rather than issues in the user's code.
func (p pass) reportInternalError(rng analysis.Range, format string, args ...any) {
	msg := []byte("Internal Error: ")
	msg = fmt.Appendf(msg, format, args...)

	p.Report(analysis.Diagnostic{Pos: rng.Pos(), End: rng.End(), Message: string(msg)})
}

// reportInvalidTarget reports an internal error if a target Init field is occupied.
func (p pass) reportInvalidTarget(init, source analysis.Range) {
	// this should never happen
	p.Report(analysis.Diagnostic{
		Pos:     init.Pos(),
		End:     init.End(),
		Message: "Internal error: Init is not empty",
		Related: []analysis.RelatedInformation{{Pos: source.Pos(), End: source.End(), Message: "Move candidate"}},
	})
}
