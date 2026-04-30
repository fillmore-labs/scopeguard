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
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("bitmask: ")

	fs := flag.CommandLine

	var o options
	o.registerFlags(fs)

	_ = fs.Parse(os.Args[1:]) // exits on error

	if len(o.typeNames) == 0 {
		fs.Usage()
		os.Exit(2)
	}

	var dir string

	switch args := fs.Args(); len(args) {
	case 0:
		dir = "."

	case 1:
		dir = args[0]

	default:
		log.Fatalf("Too many arguments: %v", args)
	}

	ctx := context.Background()

	specs, err := load(ctx, dir, o)
	if err != nil {
		log.Fatal(err)
	}

	tmpl, err := parseTemplates()
	if err != nil {
		log.Fatal(err)
	}

	if o.output != "" {
		if filepath.Ext(o.output) != ".go" {
			log.Fatalf("file %s needs a .go suffix", o.output)
		}

		// One file for every requested type, named explicitly. A single file must
		// hold a single package, so reject specs that would land in different
		// packages or mix test and non-test code.
		for _, spec := range specs[1:] {
			if spec.Package != specs[0].Package || spec.inTest != specs[0].inTest {
				log.Fatalf("cannot write to single file (-output=%q) when matching types span multiple packages", o.output)
			}
		}

		// Code for an external "_test" package (or an in-package test type) only
		// compiles in a _test.go file. The default path promotes such files via
		// outputName; with an explicit -output the caller must match.
		if specs[0].inTest && !strings.HasSuffix(o.output, "_test.go") {
			log.Fatalf("types are declared in test files; -output=%q must end with _test.go", o.output)
		}

		if err := writeFiles(filepath.Join(dir, o.output), tmpl, specs, o.json); err != nil {
			log.Fatal(err)
		}

		return
	}

	// Default: one file per type, named <type>_bitmask.go (or <type>_bitmask_test.go
	// for types declared in test files). Reject type names that lowercase to the same
	// basename, otherwise the second would silently overwrite the first.
	seen := make(map[string]string, len(specs))

	for _, spec := range specs {
		outPath := filepath.Join(dir, outputName(spec))
		if prev, ok := seen[outPath]; ok {
			log.Fatalf("types %s and %s both map to %s", prev, spec.TypeName, outPath)
		}

		seen[outPath] = spec.TypeName

		if err := writeFiles(outPath, tmpl, []Spec{spec}, o.json); err != nil {
			log.Fatal(err)
		}
	}
}

// outputName returns the default generated file name for spec: the lower-cased
// type name with a "_bitmask" suffix, promoted to a "_test.go" file when the
// type was declared in a test file.
func outputName(spec Spec) string {
	name := strings.ToLower(spec.TypeName) + "_bitmask"
	if spec.inTest {
		return name + "_test.go"
	}

	return name + ".go"
}
