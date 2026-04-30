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
	"embed"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templates embed.FS

const (
	mainTmpl   = "root.tmpl"
	jsonv2Tmpl = "jsonv2.tmpl"
)

func parseTemplates() (*template.Template, error) {
	return template.ParseFS(templates, "templates/*.tmpl")
}

// generate generates Go source code based on the provided template and specifications.
func generate(tmpl *template.Template, specs []Spec) ([]byte, error) {
	if len(specs) == 0 {
		return nil, errors.New("no specs to generate")
	}

	// templateData is the template's top-level data: one package containing one or more
	// types' generated code.
	type templateData struct {
		// Package is the generated package name.
		Package string

		// Specs are the descriptions of the generated types.
		Specs []Spec

		// UseStrconv is true if we need to import the "strconv" package.
		UseStrconv bool

		// Debug skips the generated header and formatting for diagnosing the output.
		// Set manually during development; not exposed as a flag.
		Debug bool
	}

	data := &templateData{
		Package:    specs[0].Package,
		Specs:      specs,
		UseStrconv: false,
		Debug:      false,
	}

	for _, spec := range specs {
		if spec.Package != data.Package {
			return nil, fmt.Errorf("specs span multiple packages: %s and %s", data.Package, spec.Package)
		}

		data.UseStrconv = data.UseStrconv || spec.BoolFlag
	}

	var raw bytes.Buffer
	if err := tmpl.Execute(&raw, data); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	if data.Debug {
		return raw.Bytes(), nil
	}

	src, err := format.Source(raw.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated source: %w", err)
	}

	return src, nil
}

// writeFiles writes the main generated file at path and, when json is set, the
// build-tagged jsonv2 sibling.
func writeFiles(path string, tmpl *template.Template, specs []Spec, json bool) error {
	t := tmpl.Lookup(mainTmpl)
	if err := writeFile(path, t, specs); err != nil {
		return err
	}

	if json {
		t := tmpl.Lookup(jsonv2Tmpl)
		if err := writeFile(jsonv2Path(path), t, specs); err != nil {
			return err
		}
	}

	return nil
}

func writeFile(path string, tmpl *template.Template, specs []Spec) error {
	src, err := generate(tmpl, specs)
	if err != nil {
		return fmt.Errorf("generate %s: %w", tmpl.Name(), err)
	}

	if err := os.WriteFile(path, src, 0o644); err != nil { // #nosec G306 -- generated code.
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

// jsonv2Path derives the build-tagged jsonv2 file name by inserting "_jsonv2"
// before the extension: "bitmask_bitmask.go" -> "bitmask_bitmask_jsonv2.go". For
// a test file the marker is inserted before the "_test.go" suffix so the sibling
// stays a test file: "a_bitmask_test.go" -> "a_bitmask_jsonv2_test.go".
func jsonv2Path(path string) string {
	const testSuffix = "_test.go"
	if strings.HasSuffix(path, testSuffix) {
		return path[:len(path)-len(testSuffix)] + "_jsonv2" + testSuffix
	}

	ext := filepath.Ext(path)

	return path[:len(path)-len(ext)] + "_jsonv2" + ext
}
