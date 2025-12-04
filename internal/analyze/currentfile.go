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
)

// CurrentFile holds file information for analysis.
type CurrentFile struct {
	node      *ast.File
	file      *token.File
	generated bool
}

// NewCurrentFile creates a new FileInfo from a [token.FileSet] and an *[ast.File].
func NewCurrentFile(fset *token.FileSet, aF *ast.File) CurrentFile {
	if aF == nil {
		return CurrentFile{}
	}

	tF := fset.File(aF.FileStart)
	if tF == nil {
		return CurrentFile{}
	}

	gen := ast.IsGenerated(aF)

	return CurrentFile{aF, tF, gen}
}

// Valid returns true if the FileInfo is valid.
func (c CurrentFile) Valid() bool {
	return c.file != nil
}

// Generated returns true if the file is a generated file.
func (c CurrentFile) Generated() bool {
	return c.generated
}

// Lines returns the number of Lines a statement spans.
func (c CurrentFile) Lines(stmt ast.Node) int {
	return c.line(stmt.End()) - c.line(stmt.Pos()) + 1
}

func (c CurrentFile) line(pos token.Pos) int {
	return c.file.PositionFor(pos, false).Line
}

// HasNoLintComment checks if a declaration is preceded by a //nolint:scopeguard comment.
func (c CurrentFile) HasNoLintComment(pos token.Pos) bool {
	if c.node == nil {
		return false
	}

	// find the first comment starting after the declaration
	i, _ := slices.BinarySearchFunc(c.node.Comments, pos, compareCommentPosition)
	if i >= len(c.node.Comments) {
		return false
	}

	comment := c.node.Comments[i].List[0]

	if c.line(comment.Pos()) != c.line(pos) {
		return false // not on this line
	}

	if linters, ok := parseDirective(comment.Text); !ok || !slices.Contains(linters, scopeguard) {
		return false
	}

	return true
}

func compareCommentPosition(c *ast.CommentGroup, p token.Pos) int { return int(c.Pos() - p) }

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
