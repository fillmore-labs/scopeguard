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
	"fmt"
	"go/token"
	"go/types"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Spec describes a single bitmask type ready for code generation.
type Spec struct {
	// Package is the name of the package where the generated code will reside.
	Package string

	// TypeName is the name of the type being described by the Spec.
	TypeName string

	// Receiver specifies the method receiver's name used in generated code.
	Receiver string

	// TrueValue is the identifier assigned when the input is the literal "true"
	// (in JSON or via the generated [flag.Value]). Empty disables that branch.
	TrueValue string

	// Bits holds one entry per single-bit const, indexed by bit position. The
	// position must be contiguous from 0; parse rejects gaps. This constraint
	// allows AppendText to use a dense array for fast lookups.
	Bits []constInfo

	// Aliases are line-commented multi-bit combinations, emitted in
	// source-declaration order so that AppendText can short-circuit to one and
	// UnmarshalText can accept its name as a single token. Combos without line
	// comments are runtime helpers and are silently ignored.
	Aliases []constInfo

	// BoolFlag instructs the template to emit a boolean [flag.Value] helper.
	BoolFlag bool

	// JSON instructs the template to emit MarshalJSON/UnmarshalJSON
	// methods. The marshaler emits a JSON array of bit names (one per set bit,
	// in ascending position order); the unmarshaler accepts the same form and
	// also recognizes aliases.
	JSON bool

	// inTest reports that the type was declared in a test file (an in-package
	// _test.go or an external _test package). The generated code is then routed
	// to a _test.go file so it shares the test build constraint of its source.
	inTest bool
}

// createSpecs validates each type's collected constants and builds the per-type
// Spec values ready for the template. The returned slice is ordered by the
// source position of each type's declaration, so the generated output mirrors
// the order in which the types appear in the package (stable, and independent
// of the order in which they were requested on the command line).
func createSpecs(p parser) ([]Spec, error) {
	specs := make([]Spec, 0, len(p.types))

	for tn, cd := range p.sortedTypes {
		typeName := tn.Name()

		bits, err := cd.buildBits()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", typeName, err)
		}

		aliases, err := cd.buildAliases()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", typeName, err)
		}

		// Detect any name collision across bits / aliases: parseName's switch
		// would otherwise have duplicate cases.
		if err := checkNameCollisions(bits, aliases); err != nil {
			return nil, err
		}

		spec := Spec{
			Package:  pkgName(tn),
			TypeName: typeName,
			Receiver: receiver(typeName),
			Bits:     bits,
			Aliases:  aliases,
			inTest:   testFile(p.pkg.Fset, tn),
		}

		specs = append(specs, spec)
	}

	return specs, nil
}

// pkgName returns the name of the package where tn is declared.
func pkgName(tn *types.TypeName) string {
	pkg := tn.Pkg()
	if pkg == nil {
		return ""
	}

	return pkg.Name()
}

// receiver extracts the first rune from the given typeName, converts it to lowercase, and returns it as a string.
func receiver(typeName string) string {
	rcv, _ := utf8.DecodeRuneInString(typeName)
	return string(unicode.ToLower(rcv))
}

// testFile checks if a given type's definition is in a Go test file ending with "_test.go".
func testFile(fset *token.FileSet, tn *types.TypeName) bool {
	pos := fset.PositionFor(tn.Pos(), false)
	return strings.HasSuffix(pos.Filename, "_test.go")
}

// checkNameCollisions returns an error if any two of the bit / alias names
// are the same string. The generated parseName switches on those names; a
// collision would emit duplicate cases (a compile error in the generated file).
func checkNameCollisions(bits, aliases []constInfo) error {
	owners := make(map[string]string, len(bits)+len(aliases))

	claim := func(cn constInfo) error {
		if prev, ok := owners[cn.Name]; ok {
			return fmt.Errorf("name %q is used by both %s and %s", cn.Name, prev, cn.Const)
		}

		owners[cn.Name] = cn.Const

		return nil
	}

	for _, b := range bits {
		if err := claim(b); err != nil {
			return err
		}
	}

	for _, a := range aliases {
		if err := claim(a); err != nil {
			return err
		}
	}

	return nil
}
