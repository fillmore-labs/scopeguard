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

package astutil_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	. "fillmore-labs.com/scopeguard/internal/astutil"
)

func TestAnalyzeComments(t *testing.T) {
	t.Parallel()

	tests := [...]struct {
		name        string
		src         string
		declName    string
		linter      string
		wantNameSub string
		wantSuppr   bool
	}{
		{
			name:      "leading deprecated",
			src:       "// Deprecated: gone.\ntype Foo int",
			declName:  "Foo",
			linter:    "errorname",
			wantSuppr: true,
		},
		{
			name:      "deprecated after paragraph",
			src:       "// Foo does things.\n//\n// Deprecated: gone.\ntype Foo int",
			declName:  "Foo",
			linter:    "errorname",
			wantSuppr: true,
		},
		{
			name:        "deprecated mid-line ignored",
			src:         "// Foo is not Deprecated: here.\ntype Foo int",
			declName:    "Foo",
			linter:      "errorname",
			wantNameSub: "Foo is not",
		},
		{
			name:      "nolint exact linter",
			src:       "//nolint:errorname\ntype Foo int",
			declName:  "Foo",
			linter:    "errorname",
			wantSuppr: true,
		},
		{
			name:      "nolint linter list head",
			src:       "//nolint:errorname,foo\ntype Foo int",
			declName:  "Foo",
			linter:    "errorname",
			wantSuppr: true,
		},
		{
			name:      "nolint linter list tail",
			src:       "//nolint:foo,errorname\ntype Foo int",
			declName:  "Foo",
			linter:    "errorname",
			wantSuppr: true,
		},
		{
			name:      "nolint all",
			src:       "//nolint:all\ntype Foo int",
			declName:  "Foo",
			linter:    "errorname",
			wantSuppr: true,
		},
		{
			name:     "nolint other linter",
			src:      "//nolint:gocyclo\ntype Foo int",
			declName: "Foo",
			linter:   "errorname",
		},
		{
			name:      "nolint with trailing comment",
			src:       "//nolint:errorname // explanation\ntype Foo int",
			declName:  "Foo",
			linter:    "errorname",
			wantSuppr: true,
		},
		{
			name:      "nolint mixed with text",
			src:       "// Foo does things.\n//nolint:errorname\ntype Foo int",
			declName:  "Foo",
			linter:    "errorname",
			wantSuppr: true,
		},
		{
			name:        "name prefix simple",
			src:         "// FooError wraps things.\ntype FooError int",
			declName:    "FooError",
			linter:      "errorname",
			wantNameSub: "FooError wraps",
		},
		{
			name:        "name prefix with period",
			src:         "// FooError.\n//\n// Details.\ntype FooError int",
			declName:    "FooError",
			linter:      "errorname",
			wantNameSub: "FooError.",
		},
		{
			name:        "name prefix with comma",
			src:         "// FooError, the canonical one.\ntype FooError int",
			declName:    "FooError",
			linter:      "errorname",
			wantNameSub: "FooError, the",
		},
		{
			name:     "name prefix word boundary",
			src:      "// FooErrorX does things.\ntype FooError int",
			declName: "FooError",
			linter:   "errorname",
		},
		{
			name:     "name prefix case sensitive",
			src:      "// fooError does things.\ntype FooError int",
			declName: "FooError",
			linter:   "errorname",
		},
		{
			name:        "unicode name matches",
			src:         "// ÄErr is a thing.\ntype ÄErr int",
			declName:    "ÄErr",
			linter:      "errorname",
			wantNameSub: "ÄErr is",
		},
		{
			name:     "unicode name word boundary",
			src:      "// ÄErrX is a thing.\ntype ÄErr int",
			declName: "ÄErr",
			linter:   "errorname",
		},
		{
			name:     "directive only doc",
			src:      "//go:generate stringer\ntype Foo int",
			declName: "Foo",
			linter:   "errorname",
		},
		{
			name:      "block comment with name",
			src:       "/* FooError is a thing.\n\nDeprecated: gone. */\ntype FooError int",
			declName:  "FooError",
			linter:    "errorname",
			wantSuppr: true,
		},
		{
			name:        "block comment with star markers",
			src:         "/*\n * FooError is a thing.\n */\ntype FooError int",
			declName:    "FooError",
			linter:      "errorname",
			wantNameSub: "FooError is",
		},
		{
			name:      "block comment with star markers and deprecation",
			src:       "/*\n * FooError is a thing.\n *\n * Deprecated: gone.\n */\ntype FooError int",
			declName:  "FooError",
			linter:    "errorname",
			wantSuppr: true,
		},
		{
			name:     "block comment with bare star prefix",
			src:      "/*\n*foo\n*/\ntype FooError int",
			declName: "FooError",
			linter:   "errorname",
		},
		{
			name:        "block comment with tab after star",
			src:         "/*\n *\tFooError is a thing.\n */\ntype FooError int",
			declName:    "FooError",
			linter:      "errorname",
			wantNameSub: "FooError is",
		},
		{
			name:        "empty name skips prefix check",
			src:         "// Foo does things.\ntype Foo int",
			declName:    "",
			linter:      "errorname",
			wantNameSub: "",
		},
		{
			name:        "leading blank lines",
			src:         "//\n//\n// FooError is a thing.\ntype FooError int",
			declName:    "FooError",
			linter:      "errorname",
			wantNameSub: "FooError is",
		},
		{
			name:     "different linter not suppressed",
			src:      "//nolint:errortype\ntype Foo int",
			declName: "Foo",
			linter:   "errorname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			src := "package p\n\n" + tt.src

			doc, fileStart := parseDoc(t, src)
			pos, suppressed := AnalyzeComments(doc, tt.declName, tt.linter)

			if suppressed != tt.wantSuppr {
				t.Errorf("Suppress = %v, want %v", suppressed, tt.wantSuppr)
			}

			if tt.wantNameSub == "" {
				if len(pos) > 0 {
					offset := int(pos[0] - fileStart)
					t.Errorf("NameStart = %d (offset %d, %q), want NoPos",
						pos, offset, srcAround(src, offset))
				}

				return
			}

			wantOffset := strings.Index(src, tt.wantNameSub)
			if wantOffset < 0 || len(pos) == 0 {
				t.Fatalf("substring %q not found in src", tt.wantNameSub)
			}

			if wantPos := fileStart + token.Pos(wantOffset); pos[0] != wantPos {
				gotOffset := int(pos[0] - fileStart)
				t.Errorf("NameStart offset = %d (%q), want %d (%q)",
					gotOffset, srcAround(src, gotOffset), wantOffset, tt.wantNameSub)
			}
		})
	}
}

func TestAnalyzeComments_Nil(t *testing.T) {
	t.Parallel()

	pos, suppressed := AnalyzeComments(nil, "Foo", "errorname")

	if suppressed {
		t.Errorf("Suppress = true, want false")
	}

	if len(pos) > 0 {
		t.Errorf("NameStart = %d, want NoPos", pos)
	}
}

// TestAnalyzeComments_Qualified verifies that a qualified reference such as
// "pkg.FooError" is not mistaken for a standalone mention: only the bare
// occurrence is reported, even when it is not on the leading line.
func TestAnalyzeComments_Qualified(t *testing.T) {
	t.Parallel()

	const src = `package p

// FooError does things.
// It mirrors errpkg.FooError but is local.
type FooError int`

	doc, fileStart := parseDoc(t, src)

	pos, suppressed := AnalyzeComments(doc, "FooError", "errorname")
	if suppressed {
		t.Fatalf("Suppress = true, want false")
	}

	wantOffset := strings.Index(src, "FooError does")

	if len(pos) != 1 {
		got := make([]int, len(pos))
		for i, p := range pos {
			got[i] = int(p - fileStart)
		}

		t.Fatalf("got %d mentions at offsets %v, want 1 (the bare FooError only)", len(pos), got)
	}

	if wantPos := fileStart + token.Pos(wantOffset); pos[0] != wantPos {
		t.Errorf("mention offset = %d, want %d", int(pos[0]-fileStart), wantOffset)
	}
}

// parseDoc parses src and returns the doc comment of the first top-level
// declaration. The returned offset is the start position of src in the file's
// token.Pos space, useful to translate absolute positions back to offsets.
func parseDoc(t *testing.T, src string) (*ast.CommentGroup, token.Pos) {
	t.Helper()

	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, "p.go", src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(f.Decls) == 0 {
		t.Fatalf("no declarations parsed")
	}

	gdec, ok := f.Decls[0].(*ast.GenDecl)
	if !ok {
		t.Fatalf("first declaration is %T, want *ast.GenDecl", f.Decls[0])
	}

	return gdec.Doc, f.FileStart
}

func srcAround(src string, offset int) string {
	if offset < 0 || offset >= len(src) {
		return "<out of range>"
	}

	end := min(offset+10, len(src))

	return src[offset:end]
}
