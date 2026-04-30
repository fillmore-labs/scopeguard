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

package engine

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"

	"fillmore-labs.com/scopeguard/internal/run"
	"fillmore-labs.com/scopeguard/internal/set"
	"fillmore-labs.com/scopeguard/internal/typeutil"
)

var (
	// ErrUnmatchedFunctions is returned when a filter doesn't match a function.
	ErrUnmatchedFunctions = errors.New("unmatched functions")

	// ErrPackageLoad is returned when loaded packages contain errors.
	ErrPackageLoad = errors.New("package load errors")
)

// AnalyzePackages analyzes specified Go packages. It requires an absolute path for the package directory.
func AnalyzePackages(ctx context.Context, args Args, a *analysis.Analyzer) (*checker.Graph, error) {
	pkgs, err := loadPackages(ctx, args)
	if err != nil {
		return nil, err
	}

	graph, err := checker.Analyze([]*analysis.Analyzer{a}, pkgs, nil)
	if err != nil {
		return nil, fmt.Errorf("can't analyze packages: %w", err)
	}

	if err := checkMatched(graph, args.Functions); err != nil {
		return nil, err
	}

	return graph, nil
}

func loadPackages(ctx context.Context, args Args) ([]*packages.Package, error) {
	if !filepath.IsAbs(args.Dir) {
		return nil, fmt.Errorf("not an absolute path: %q", args.Dir)
	}

	cfg := &packages.Config{
		Mode:    packages.LoadAllSyntax,
		Context: ctx,
		Dir:     args.Dir,
		ParseFile: func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
			const mode = parser.AllErrors | parser.ParseComments | parser.SkipObjectResolution
			return parser.ParseFile(fset, filename, src, mode)
		},
		Tests: true,
	}

	pkgs, err := packages.Load(cfg, args.Packages...)
	if err != nil { // failure to enumerate package
		return nil, fmt.Errorf("can't read packages in %q: %w", args.Dir, err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("%w: patterns %s do not match any package", ErrPackageLoad, strings.Join(args.Packages, ", "))
	}

	if err := checkPackageErrors(pkgs); err != nil {
		return nil, err
	}

	return pkgs, nil
}

// checkPackageErrors returns an error aggregating any load/parse/type errors
// attached to the initial packages.
func checkPackageErrors(pkgs []*packages.Package) error {
	var errs []error

	for _, pkg := range pkgs {
		for _, pkgErr := range pkg.Errors {
			errs = append(errs, fmt.Errorf("%s: %w", pkg.PkgPath, pkgErr))
		}
	}

	if len(errs) == 0 {
		return nil
	}

	const maxErrors = 5
	if len(errs) > maxErrors {
		errs = append(errs[:maxErrors], fmt.Errorf("... and %d more errors", len(errs)-maxErrors))
	}

	return fmt.Errorf("%w: %w", ErrPackageLoad, errors.Join(errs...))
}

func checkMatched(graph *checker.Graph, functions []typeutil.LocalFuncName) error {
	processed, err := collectProcessed(graph)
	if err != nil {
		return err
	}

	var unmatched []string

	for _, f := range functions {
		if !processed.Contains(f) {
			unmatched = append(unmatched, f.String())
		}
	}

	if len(unmatched) > 0 {
		return fmt.Errorf("%w: %s", ErrUnmatchedFunctions, strings.Join(unmatched, ", "))
	}

	return nil
}

func collectProcessed(graph *checker.Graph) (set.Set[typeutil.LocalFuncName], error) {
	matched := set.New[typeutil.LocalFuncName]()

	for _, act := range graph.Roots {
		if err := act.Err; err != nil {
			return nil, fmt.Errorf("analyze failed in %q: %w", act.Package.PkgPath, err)
		}

		res := act.Result.(run.Result)
		for _, p := range res.Processed {
			matched.Add(p.Name)
		}
	}

	return matched, nil
}
