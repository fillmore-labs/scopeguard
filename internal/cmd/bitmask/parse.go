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
	"cmp"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"maps"
	"slices"
	"strings"
	"unicode"

	"golang.org/x/tools/go/packages"

	"fillmore-labs.com/scopeguard/internal/typeutil"
)

// load parses dir and returns one Spec per requested type name. It drives the pipeline:
// load the package (including its test variants), resolve the requested types, collect
// their line-commented constants, build and validate the specs, and overlay the
// command-line options.
func load(ctx context.Context, dir string, o options) ([]Spec, error) {
	cfg := &packages.Config{
		Mode:    packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Context: ctx,
		Dir:     dir,
		Tests:   true,
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, fmt.Errorf("load package: %w", err)
	}

	// Packages may carry compile errors elsewhere; we only need the requested types
	// themselves to be sound. lookupTypes will report a clear error if a requested
	// type is missing or has the wrong underlying kind.
	variants := typedVariants(pkgs)
	if len(variants) == 0 {
		// Unrecoverable: no type info at all. Surface package errors and bail.
		packages.PrintErrors(pkgs)

		return nil, fmt.Errorf("no type information for any package in %s", dir)
	}

	specs, err := parseVariants(variants, o.typeNames)
	if err != nil {
		return nil, err
	}

	if err := applyFlags(specs, o); err != nil {
		return nil, err
	}

	return specs, nil
}

// typedVariants selects the package variants that carry type information and
// returns one authoritative variant per package name, ordered by name.
//
// With Tests enabled, packages.Load yields up to four variants of a directory:
// the package itself, the same package compiled with its in-package _test.go
// files, the external "_test" package, and the synthetic test-main. The
// test-compiled variant is a superset of the plain one (it adds the in-package
// test files), so for each package name we keep the variant with the most
// syntax files and drop the rest. The synthetic test-main (its import path ends
// in ".test") never declares user types and is skipped.
func typedVariants(pkgs []*packages.Package) []*packages.Package {
	best := make(map[string]*packages.Package, len(pkgs))
	for _, pkg := range pkgs {
		if pkg.Types == nil || len(pkg.Syntax) == 0 || strings.HasSuffix(pkg.PkgPath, ".test") {
			continue
		}

		name := pkg.Types.Name()
		if cur, ok := best[name]; !ok || len(pkg.Syntax) > len(cur.Syntax) {
			best[name] = pkg
		}
	}

	names := slices.Sorted(maps.Keys(best))

	variants := make([]*packages.Package, len(names))
	for i, name := range names {
		variants[i] = best[name]
	}

	return variants
}

// parser holds the state during constant extraction.
//
// It can be used as a value type and mutated, since types is a reference.
type parser struct {
	pkg   *packages.Package
	types map[*types.Named]*typeConstants
}

// parseVariants resolves the requested type names across the package variants
// and builds one Spec per type. A type must resolve in exactly one output
// package: one declared in both the package and its external "_test" package is
// rejected as ambiguous. Names that resolve nowhere are reported as not found.
func parseVariants(variants []*packages.Package, typeNames []string) ([]Spec, error) {
	var specs []Spec

	owner := make(map[string]string, len(typeNames)) // type name -> package that claimed it

	remaining := make(map[string]struct{}, len(typeNames))
	for _, name := range typeNames {
		remaining[name] = struct{}{}
	}

	for _, pkg := range variants {
		p := parser{
			pkg:   pkg,
			types: make(map[*types.Named]*typeConstants, len(typeNames)),
		}

		found, err := p.lookupTypes(typeNames)
		if err != nil {
			return nil, err
		}

		if len(found) == 0 {
			continue
		}

		pkgName := pkg.Types.Name()

		for _, name := range found {
			if prev, ok := owner[name]; ok {
				return nil, fmt.Errorf(
					"type %s: declared in both package %s and package %s; generate them separately",
					name, prev, pkgName,
				)
			}

			owner[name] = pkgName
			delete(remaining, name)
		}

		for decl := range typeutil.AllConstDecls(pkg.Syntax) {
			if err := p.collectConstants(decl); err != nil {
				return nil, err
			}
		}

		// [parser.collectConstants] walks [pkg.Syntax] which is sorted by file path and
		// can diverge from global source position when bitmask consts span multiple files.
		for _, typ := range p.types {
			typ.sortCombos()
		}

		pkgSpecs, err := createSpecs(p)
		if err != nil {
			return nil, err
		}

		specs = append(specs, pkgSpecs...)
	}

	if len(remaining) > 0 {
		return nil, fmt.Errorf("type(s) not found: %s", strings.Join(slices.Sorted(maps.Keys(remaining)), ", "))
	}

	return specs, nil
}

// lookupTypes resolves the requested type names that are declared in this
// package variant to their *[types.Named] and pre-allocates a [typeConstants]
// bucket per type, returning the names it resolved. Names not declared here are
// skipped (they may belong to another variant). Two requested names that
// resolve to the same defined type (e.g. an alias and its base, or repeated
// names) are rejected: they would emit duplicate method definitions.
func (p parser) lookupTypes(typeNames []string) ([]string, error) {
	seen := make(map[*types.Named]string, len(typeNames))
	found := make([]string, 0, len(typeNames))

	pkgScope := p.pkg.Types.Scope()
	for _, typeName := range typeNames {
		obj := pkgScope.Lookup(typeName)
		if obj == nil {
			continue // not declared in this package variant
		}

		named, err := resolve(obj)
		if err != nil {
			return nil, fmt.Errorf("type %s: %w", typeName, err)
		}

		if prev, ok := seen[named]; ok {
			if prev == typeName {
				return nil, fmt.Errorf("duplicate type %s", typeName)
			}

			return nil, fmt.Errorf("%s and %s refer to the same defined type", prev, typeName)
		}

		p.types[named] = &typeConstants{single: make(map[int]constInfo)}
		seen[named] = typeName
		found = append(found, typeName)
	}

	return found, nil
}

// resolve resolves obj and verifies that its underlying type is
// an integer type suitable for use as a bitmask.
//
// Aliases are followed to their base type when the base lives in the current package.
func resolve(obj types.Object) (*types.Named, error) {
	tn, ok := obj.(*types.TypeName)
	if !ok {
		return nil, errors.New("is not a type")
	}

	named, ok := types.Unalias(tn.Type()).(*types.Named)
	if !ok {
		return nil, errors.New("is not a named type")
	}

	if namedObj := named.Obj(); namedObj != tn && namedObj.Pkg() != tn.Pkg() {
		pkgPath := "<universe>"
		if p := namedObj.Pkg(); p != nil {
			pkgPath = p.Path()
		}

		return nil, fmt.Errorf("is an alias for %s.%s; generate against the original type", pkgPath, namedObj.Name())
	}

	if basic, ok := named.Underlying().(*types.Basic); !ok || basic.Info()&types.IsInteger == 0 {
		return nil, errors.New("underlying type is not an integer")
	}

	return named, nil
}

// collectConstants walks every line-commented constant declaration and
// routes those belonging to a requested type into its typeConstants bucket. Constants
// of other types, and uncommented constants (internal helpers), are skipped.
func (p parser) collectConstants(decl *ast.ValueSpec) error {
	for _, id := range decl.Names {
		def, ok := p.pkg.TypesInfo.Defs[id].(*types.Const)
		if !ok {
			continue
		}

		tn, ok := def.Type().(*types.Named)
		if !ok {
			continue
		}

		ctyp, ok := p.types[tn]
		if !ok {
			continue // Not a type we are interested in
		}

		comment := strings.TrimSpace(decl.Comment.Text())
		if comment == "" {
			continue // No line comment, internal helper
		}

		// UnmarshalText splits its input on ',' and space; a name containing either would not round-trip.
		if strings.ContainsFunc(comment, func(r rune) bool { return r == ',' || unicode.IsSpace(r) }) {
			return fmt.Errorf("const %s name %q contains comma or whitespace", id.Name, comment)
		}

		value, exact := uint64Val(def.Val())
		if !exact {
			return fmt.Errorf("const %s: value %s not representable as uint64", id.Name, def)
		}

		cd := constInfo{Const: id.Name, pos: id.Pos(), Name: comment, value: value}
		if err := ctyp.classify(cd); err != nil {
			return err
		}
	}

	return nil
}

func uint64Val(x constant.Value) (uint64, bool) {
	if x.Kind() != constant.Int {
		return 0, false
	}

	return constant.Uint64Val(x)
}

// sortedTypes is an iterator over the *[types.TypeName] ordered by the source position
// of each type's declaration, so error messages and generated output are stable
// across runs.
func (p parser) sortedTypes(yield func(*types.TypeName, *typeConstants) bool) {
	namedTypes := make([]*types.Named, 0, len(p.types))
	for named := range p.types {
		namedTypes = append(namedTypes, named)
	}

	slices.SortFunc(namedTypes, func(a, b *types.Named) int { return cmp.Compare(a.Obj().Pos(), b.Obj().Pos()) })

	for _, named := range namedTypes {
		if !yield(named.Obj(), p.types[named]) {
			break
		}
	}
}
