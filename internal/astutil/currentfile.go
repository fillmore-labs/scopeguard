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
	"go/ast"
	"go/token"
	"regexp"
	"slices"
	"strings"
)

// scopeguard is the name of the linter.
const scopeguard = "scopeguard"

// CurrentFile holds file information for analysis.
type CurrentFile struct {
	file      *ast.File
	handle    *token.File
	generated bool
}

// NewCurrentFile creates a new [CurrentFile] from a [token.FileSet] and an *[ast.File].
func NewCurrentFile(fset *token.FileSet, file *ast.File) CurrentFile {
	if file == nil {
		return CurrentFile{}
	}

	handle := fset.File(file.FileStart)
	if handle == nil {
		return CurrentFile{}
	}

	generated := ast.IsGenerated(file)

	return CurrentFile{file, handle, generated}
}

// Valid returns true if the [CurrentFile] was successfully created
// from a valid file handle.
func (c CurrentFile) Valid() bool {
	return c.handle != nil
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
	return c.handle.PositionFor(pos, false).Line
}

// NoLintComment checks if a line is followed by a //nolint:scopeguard comment.
func (c CurrentFile) NoLintComment(pos token.Pos) bool {
	if c.file == nil {
		return false
	}

	// find the first comment starting after the declaration
	i, _ := slices.BinarySearchFunc(c.file.Comments, pos,
		func(c *ast.CommentGroup, p token.Pos) int { return int(c.Pos() - p) })
	if i >= len(c.file.Comments) {
		return false
	}

	comment := c.file.Comments[i].List[0]

	if c.line(comment.Pos()) != c.line(pos) {
		return false // not on this line
	}

	return CommentHasNoLint(comment)
}

var nolintPattern = regexp.MustCompile(`^//\s*nolint:([a-zA-Z0-9,_-]+)`)

// CommentHasNoLint checks if the provided comment contains a `//nolint:scopeguard` directive.
func CommentHasNoLint(comment *ast.Comment) bool {
	matches := nolintPattern.FindStringSubmatch(comment.Text)
	if matches == nil {
		return false
	}

	// Parse comma-separated linter list
	for linter := range strings.SplitSeq(matches[1], ",") {
		if l := strings.ToLower(strings.TrimSpace(linter)); l == scopeguard || l == "all" {
			return true
		}
	}

	return false
}
