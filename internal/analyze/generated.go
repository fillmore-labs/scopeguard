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

import "go/ast"

// generatedChecker determines whether files are generated.
//
// It uses a nil-pattern optimization and lazy caching:
//   - nil checker: Always returns false (treats all files as non-generated)
//   - Non-nil checker: Lazily checks and caches each file's generated status
type generatedChecker map[*ast.File]bool

// isGenerated reports whether the given file is generated.
//
// When the checker is nil, it always returns false (all files are treated as non-generated).
// When non-nil, it checks using [ast.IsGenerated] and caches the result for subsequent calls.
func (g generatedChecker) isGenerated(f *ast.File) bool {
	if g == nil {
		return false
	}

	isGenerated, seen := g[f]

	if !seen {
		isGenerated = ast.IsGenerated(f)
		g[f] = isGenerated
	}

	return isGenerated
}
