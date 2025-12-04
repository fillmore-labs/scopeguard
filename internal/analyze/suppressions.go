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
	"go/ast"
	"go/token"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/tools/go/ast/inspector"
)

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

func enclosingFile(c inspector.Cursor) *ast.File {
	for e := range c.Enclosing((*ast.File)(nil)) {
		f, _ := e.Node().(*ast.File)

		return f
	}

	return nil
}

type lineFinder struct{ *token.File }

func (lf lineFinder) line(pos token.Pos) int {
	return lf.PositionFor(pos, false).Line
}

// lines returns the number of lines a statement spans.
func (lf lineFinder) lines(stmt ast.Node) int {
	return lf.line(stmt.End()) - lf.line(stmt.Pos()) + 1
}

// hasNoLintComment checks if a declaration is preceded by a //nolint:scopeguard comment.
func (lf lineFinder) hasNoLintComment(file *ast.File, declNode ast.Node) bool {
	for _, co := range file.Comments {
		if co.Pos() > declNode.Pos() {
			if lf.line(co.Pos()) != lf.line(declNode.Pos()) {
				break
			}

			if linters, ok := parseDirective(co.List[0].Text); !ok || !slices.Contains(linters, scopeguard) {
				break
			}

			return true
		}
	}

	return false
}

var nolintPattern = regexp.MustCompile(`^//\s*nolint:([a-zA-Z0-9,_-]+)`)

// parseDirective extracts linter names from a nolint comment.
func parseDirective(text string) (linters []string, ok bool) {
	matches := nolintPattern.FindStringSubmatch(text)
	if matches == nil {
		return nil, false
	}

	// Parse comma-separated linter list
	linters = strings.Split(matches[1], ",")
	for i, l := range linters {
		linters[i] = strings.ToLower(strings.TrimSpace(l))
	}

	return linters, true
}

// scopeguard is the name of the linter.
const scopeguard = "scopeguard"
