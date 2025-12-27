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

package astutil

import (
	"fmt"

	"golang.org/x/tools/go/analysis"
)

// InternalError reports an internal error diagnostic.
// These errors indicate bugs in the analyzer logic rather than issues in the user's code.
func InternalError(p *analysis.Pass, rng analysis.Range, format string, args ...any) {
	msg := []byte("Internal Error: ")
	msg = fmt.Appendf(msg, format, args...)

	p.Report(analysis.Diagnostic{Pos: rng.Pos(), End: rng.End(), Message: string(msg)})
}
