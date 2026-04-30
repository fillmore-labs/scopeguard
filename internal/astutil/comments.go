// Copyright 2026 Oliver Eikemeier. All Rights Reserved.
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
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// AnalyzeComments walks doc once and reports whether the diagnostic must be
// suppressed because doc carries a [deprecation notice] or a nolint directive
// matching linter, and, when not suppressed, the positions of every standalone
// mention of name on any non-blank comment line. Qualified references such as
// "pkg.name" are skipped. Pass an empty name to skip the mention search.
//
// [deprecation notice]: https://go.dev/wiki/Deprecated
func AnalyzeComments(doc *ast.CommentGroup, name, linter string) (mentions []token.Pos, suppressed bool) {
	if doc == nil {
		return nil, false
	}

	state := lineState{name: name}

	for _, cmt := range doc.List {
		if len(cmt.Text) < 2 {
			continue
		}

		switch bodyStart := cmt.Slash + 2; cmt.Text[1] {
		case '/': // "//"-style: a single line.
			if d, ok := ParseDirective(cmt.Slash, cmt.Text); ok {
				if isNoLint(d, linter) {
					return nil, true
				}

				continue // Directives are invisible to godoc, so we don't feed them to state.
			}

			line := cmt.Text[2:]
			pos := bodyStart

			// "//"-style trims a single leading space per godoc convention
			if len(line) > 0 && line[0] == ' ' {
				line = line[1:]
				pos++
			}

			if !state.processLine(line, pos) {
				return nil, true
			}

		case '*': // "/*"-style: may span multiple lines.
			body := cmt.Text[2 : len(cmt.Text)-2]

			for offset := 0; offset < len(body); {
				pos := token.Pos(int(bodyStart) + offset)

				line := body[offset:]

				nl := strings.IndexByte(line, '\n')
				if nl >= 0 {
					line = line[:nl]
					offset += nl + 1
				}

				line, pos = trimBlockPrefix(line, pos)

				if !state.processLine(line, pos) {
					return nil, true
				}

				if nl < 0 {
					break
				}
			}
		}
	}

	return state.mentions, false
}

// HasNoLint walks doc once and reports whether the analysis should be
// suppressed because doc carries a nolint directive matching linter.
func HasNoLint(doc *ast.CommentGroup, linter string) bool {
	return doc != nil &&
		slices.ContainsFunc(doc.List, func(cmt *ast.Comment) bool {
			return IsNoLintComment(cmt, linter)
		})
}

// HasNoLintComment checks if a line is followed by a //nolint: directive.
func HasNoLintComment(fset *token.FileSet, file *ast.File, pos token.Pos, linter string) bool {
	// find the first comment starting after the declaration
	i, _ := slices.BinarySearchFunc(file.Comments, pos,
		func(doc *ast.CommentGroup, p token.Pos) int { return int(doc.Pos() - p) })
	if i >= len(file.Comments) {
		return false
	}

	comment := file.Comments[i].List[0]

	tokenFile := fset.File(file.FileStart)

	if line, commentLine := tokenFile.PositionFor(pos, false).Line, tokenFile.PositionFor(comment.Pos(), false).Line; commentLine != line {
		return false // not on this line
	}

	return IsNoLintComment(comment, linter)
}

// IsNoLintComment checks whether cmt carries a nolint directive matching linter.
func IsNoLintComment(cmt *ast.Comment, linter string) bool {
	d, ok := ParseDirective(cmt.Slash, cmt.Text)
	return ok && isNoLint(d, linter)
}

// trimBlockPrefix trims leading whitespace from a "/*"-style comment line, then
// optionally a single "*" continuation marker, returning the adjusted line and position.
//
// Runs of whitespace are trimmed because aligned indentation is common inside block comments.
func trimBlockPrefix(line string, pos token.Pos) (string, token.Pos) {
	for len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
		line, pos = line[1:], pos+1
	}

	if len(line) > 0 && line[0] == '*' {
		switch {
		case len(line) == 1:
			line, pos = "", pos+1

		case line[1] == ' ' || line[1] == '\t':
			line, pos = line[2:], pos+2
		}
	}

	return line, pos
}

const deprecatedPrefix = "Deprecated:"

// lineState carries the per-line bookkeeping for [AnalyzeComments] across
// both single-line and block comments in a [ast.CommentGroup].
type lineState struct {
	// name is the declared identifier to locate in the comment, or "" to skip.
	name string

	// mentions holds the position of every standalone mention of name.
	mentions []token.Pos

	// prevLine is true when we have text on the previous line. It starts false, so a deprecation notice on the first line is recognized.
	prevLine bool
}

// processLine updates s from one trimmed comment line at file position pos.
// It returns true when processing should continue, and false when the line
// starts a deprecation paragraph and the enclosing diagnostic must be
// suppressed.
func (s *lineState) processLine(line string, pos token.Pos) bool {
	if blankLine(line) {
		s.prevLine = false
		return true
	}

	if s.name != "" {
		s.mentions = append(s.mentions, findMentions(line, pos, s.name)...)
	}

	if s.prevLine {
		return true
	}

	s.prevLine = true

	return !strings.HasPrefix(line, deprecatedPrefix)
}

// blankLine reports whether s contains only spaces, tabs, and carriage returns.
func blankLine(s string) bool {
	for i := range len(s) {
		switch s[i] {
		case ' ', '\t', '\r':
		default:
			return false
		}
	}

	return true
}

// findMentions returns the file position of every occurrence of name in line
// that stands alone as an identifier, so doc-comment references can be renamed
// alongside the declaration. Boundaries are Unicode-aware: a match is rejected
// when an identifier rune abuts it on either side. Qualified references such as
// "pkg.name", which denote a different symbol, are also rejected. Pass an empty
// name to skip.
func findMentions(line string, linePos token.Pos, name string) []token.Pos {
	var positions []token.Pos

	for offset := 0; offset < len(line); {
		i := strings.Index(line[offset:], name)
		if i < 0 {
			break
		}

		start := offset + i

		end := start + len(name)
		if !identRuneBefore(line, start) && !identRuneAfter(line, end) {
			positions = append(positions, linePos+token.Pos(start))
		}

		offset = end
	}

	return positions
}

// identRuneBefore reports whether the rune ending at index i in s is part of an
// identifier.
func identRuneBefore(s string, i int) bool {
	if i == 0 {
		return false
	}

	if b := s[i-1]; b < utf8.RuneSelf {
		// A leading "." marks a qualified name or selector ("pkg.name", "x.name"),
		// which refers to another symbol and must not be renamed.
		return b == '.' || isIdentASCII(b)
	}

	r, _ := utf8.DecodeLastRuneInString(s[:i])

	return isIdentRune(r)
}

// identRuneAfter reports whether the rune starting at index i in s is part of an
// identifier.
func identRuneAfter(s string, i int) bool {
	if i >= len(s) {
		return false
	}

	if b := s[i]; b < utf8.RuneSelf {
		return isIdentASCII(b)
	}

	r, _ := utf8.DecodeRuneInString(s[i:])

	return isIdentRune(r)
}

func isIdentASCII(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func isIdentRune(r rune) bool {
	if r < utf8.RuneSelf {
		return isIdentASCII(byte(r)) // #nosec G115 -- checked above.
	}

	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// isNoLint reports whether the comma-separated linter list from a
// nolint directive disables `linter`.
func isNoLint(d Directive, linter string) bool {
	return d.Tool == "nolint" && hasLinter(d.Name, linter)
}

func hasLinter(list, linter string) bool {
	if list == "" || linter == "" {
		return false
	}

	for item := range strings.SplitSeq(list, ",") {
		// trim whitespace around items (e.g. "foo, bar") before comparing.
		item = strings.TrimSpace(item)
		if item == linter || item == "all" {
			return true
		}
	}

	return false
}
