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

package main

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"text/template"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestGenerate copies testdata to a temporary directory, adjusts the replace
// directive in go.mod, runs go generate and executes the correctness tests to
// validate the generated code.
func TestGenerate(t *testing.T) {
	t.Parallel()

	testdataDir := os.DirFS(analysistest.TestData())

	// Copy testdata into tempDir
	tempDir := t.TempDir()
	if err := os.CopyFS(tempDir, testdataDir); err != nil {
		t.Fatalf("failed to copy testdata to tempDir: %v", err)
	}

	const subDir = "bitmask"

	// Remove the existing generated files in tempDir to ensure they are actually
	// recreated by go generate. flag_bitmask_test.go covers a type declared in a
	// test file, whose helpers must be regenerated into a _test.go sibling.
	generatedFiles := removeGenerated(t, tempDir, subDir)

	// This will fail on bazel
	absRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("failed to get absolute path to module root: %v", err)
	}

	replace := "fillmore-labs.com/scopeguard=" + absRoot

	goCommands := [...]struct {
		name string
		args []string
	}{
		{"mod edit", []string{"mod", "edit", "-replace", replace}},
		{"generate", []string{"generate", "./" + subDir}},
		{"test", []string{"test", "./" + subDir}},
	}

	for _, c := range goCommands {
		cmd := exec.Command("go", c.args...) // #nosec G204 -- audited, see above.
		cmd.Dir = tempDir

		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s failed: %v\noutput: %s", c.name, err, string(out))
		}
	}

	// Verify that the generated files now exist
	for _, generatedFile := range generatedFiles {
		if _, err := os.Stat(generatedFile); err != nil {
			t.Errorf("generated file was not created: %v", err)
		}
	}
}

func removeGenerated(tb testing.TB, tempDir, subDir string) []string {
	tb.Helper()

	tmp := os.DirFS(tempDir)

	dir, err := fs.ReadDir(tmp, subDir)
	if err != nil {
		tb.Fatalf("failed to read %s: %v", subDir, err)
	}

	var generatedFiles []string

	for _, d := range dir {
		name := d.Name()
		if !d.Type().IsRegular() || !strings.Contains(name, "_bitmask") || !strings.HasSuffix(d.Name(), ".go") {
			continue
		}

		path := filepath.Join(tempDir, subDir, name)
		generatedFiles = append(generatedFiles, path)

		if err := os.Remove(path); err != nil {
			tb.Fatalf("failed to remove generated files: %v", err)
		}
	}

	return generatedFiles
}

// TestGenerate_MultipleSpecs verifies that generating from multiple specs produces
// a single file with one package declaration, one import block, and one body per
// spec instead of silently overwrite earlier types' generated code.
func TestGenerate_MultipleSpecs(t *testing.T) {
	t.Parallel()

	specs := []Spec{
		{
			Package:  "demo",
			TypeName: "Alpha",
			Receiver: "a",
			Bits:     []constInfo{{Const: "AlphaOne", Name: "one"}},
		},
		{
			Package:  "demo",
			TypeName: "Beta",
			Receiver: "b",
			Bits:     []constInfo{{Const: "BetaRed", Name: "red"}, {Const: "BetaBlue", Name: "blue"}},
		},
	}

	tmpl := mainTemplate(t)

	src, err := generate(tmpl, specs)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	got := string(src)

	if n := strings.Count(got, "\npackage demo\n"); n != 1 {
		t.Errorf("expected exactly one package declaration, got %d", n)
	}

	if n := bytes.Count(src, []byte("\nimport (\n")); n != 1 {
		t.Errorf("expected exactly one import block, got %d", n)
	}

	for _, want := range []string{"_AlphaNames", "_BetaNames", "func (a Alpha) String()", "func (b Beta) String()"} {
		if !strings.Contains(got, want) {
			t.Errorf("got %q, missing %q", got, want)
		}
	}
}

func TestGenerate_RejectsCrossPackage(t *testing.T) {
	t.Parallel()

	tmpl := mainTemplate(t)

	if _, err := generate(tmpl, []Spec{
		{Package: "a", TypeName: "T1", Receiver: "t", Bits: []constInfo{{Const: "X", Name: "x"}}},
		{Package: "b", TypeName: "T2", Receiver: "t", Bits: []constInfo{{Const: "Y", Name: "y"}}},
	}); err == nil {
		t.Fatal("expected error for specs spanning multiple packages")
	}
}

// TestGenerate_TrueValue verifies that TrueValue conditionally overrides the identifier assigned for the literal "true".
func TestGenerate_TrueValue(t *testing.T) {
	t.Parallel()

	tmpl := mainTemplate(t)

	spec := Spec{
		Package:  "demo",
		TypeName: "Alpha",
		Receiver: "a",
		Bits:     []constInfo{{Const: "AlphaOne", Name: "one"}, {Const: "AlphaTwo", Name: "two"}},
		JSON:     true,
	}

	// Without TrueValue (default)
	spec.TrueValue = ""

	srcDefault, err := generate(tmpl, []Spec{spec})
	if err != nil {
		t.Fatalf("generate default: %v", err)
	}

	gotDefault := string(srcDefault)
	if strings.Contains(gotDefault, `, true)`) {
		t.Error("expected default generated code not to pass hasTrue=true to bitmask.UnmarshalJSON")
	}

	if strings.Contains(gotDefault, `bool:`) {
		t.Error("expected default generated code doc comment not to mention bool unmarshaling")
	}

	// With TrueValue set to "All"
	spec.TrueValue = "All"
	spec.Aliases = []constInfo{{Const: "All", Name: "all"}}

	srcOverride, err := generate(tmpl, []Spec{spec})
	if err != nil {
		t.Fatalf("generate override: %v", err)
	}

	if gotOverride := string(srcOverride); !strings.Contains(gotOverride, "uint64(All), true") {
		t.Errorf("expected overridden generated code to pass uint64(All), true as trueValue, got:\n%s", gotOverride)
	}
}

// TestGenerate_BoolFlag verifies that BoolFlag option correctly emits BoolFlag flag.Value implementation.
func TestGenerate_BoolFlag(t *testing.T) {
	t.Parallel()

	tmpl := mainTemplate(t)

	spec := Spec{
		Package:  "demo",
		TypeName: "Alpha",
		Receiver: "a",
		Bits:     []constInfo{{Const: "AlphaOne", Name: "one"}},
		BoolFlag: true,
	}

	src, err := generate(tmpl, []Spec{spec})
	if err != nil {
		t.Fatalf("generate with BoolFlag: %v", err)
	}
	got := string(src)

	for _, want := range []string{"func (a *Alpha) BoolFlag(mask Alpha) AlphaBoolFlag", "type AlphaBoolFlag struct", "func (v AlphaBoolFlag) String() string", "func (v AlphaBoolFlag) Set(s string) error", "func (AlphaBoolFlag) IsBoolFlag() bool"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected generated BoolFlag code to contain %q", want)
		}
	}
}

var parseOnce = sync.OnceValues(parseTemplates)

func mainTemplate(tb testing.TB) *template.Template {
	tb.Helper()

	tmpl, err := parseOnce()
	if err != nil {
		tb.Fatalf("template: %v", err)
	}

	tmpl = tmpl.Lookup(mainTmpl)
	if tmpl == nil {
		tb.Fatalf("template %s not found", mainTmpl)
	}

	return tmpl
}
