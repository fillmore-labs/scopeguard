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
	"context"
	"go/ast"
	"go/token"
	"go/types"
	"iter"
	"runtime/trace"

	"golang.org/x/tools/go/ast/edge"
	"golang.org/x/tools/go/ast/inspector"
)

// declarations collects all movable variable declarations in the analyzed code.
//
// This is Stage 1 of the analyzer pipeline. It traverses the AST looking for:
//  1. Short variable declarations (x := 1 or x, y := 1, 2)
//  2. Var statements (var x int = 1)
//  3. Named return values (func f() (x int))
//
// Named return values are special: they're implicitly available throughout the
// function body - bare return statements can use named returns anywhere.
// Any redeclarations of named returns (e.g., x, err := foo()) are recorded,
// allowing Stage 2 to filter them out.
//
// Parameters:
//   - includeGenerated: When true, analyzes generated files. When false, skips them entirely.
//
// Returns:
//   - declResult: Map of variables to their declaration indices
func (p pass) declarations(ctx context.Context, in *inspector.Inspector, includeGenerated bool) (declResult, error) {
	defer trace.StartRegion(ctx, "declarations").End()

	v := declVisitor{
		pass: p,
		result: declResult{
			decls:      make(map[*types.Var][]nodeIndex),
			notMovable: make(map[nodeIndex]struct{}),
		},
		namedReturn: make(map[*types.Var]struct{}),
		generated:   includeGenerated,
	}

	in.Root().Inspect(
		[]ast.Node{
			// keep-sorted start
			(*ast.AssignStmt)(nil),
			(*ast.DeclStmt)(nil),
			(*ast.File)(nil),
			(*ast.FuncDecl)(nil),
			(*ast.FuncType)(nil),
			// keep-sorted end
		},
		v.visit,
	)

	return v.result, nil
}

type declVisitor struct {
	pass
	result declResult

	// Temporary set to identify named return variables during processing.
	// Used to detect and mark their redeclarations in result.notMovable.
	// Named returns themselves are NOT added to result.decls since they're
	// part of the function signature and cannot be moved.
	namedReturn map[*types.Var]struct{}

	generated bool
}

func (d declVisitor) visit(c inspector.Cursor) (descend bool) {
	switch n := c.Node().(type) {
	// keep-sorted start newline_separated=yes
	case *ast.AssignStmt:
		if n.Tok != token.DEFINE {
			break // Not a short variable declaration
		}

		if kind, _ := c.ParentEdge(); kind == edge.CommClause_Comm {
			break // Don't consider short declarations in select cases
		}

		index := c.Index()

		for v, def := range d.allVars(allAssigned(n)) {
			switch def {
			case varDefined:
				// Variable definition
				if len(d.result.decls[v]) > 0 {
					d.reportInternalError(n, "Re-definition of variable %s", v.Name())
					continue
				}

				d.result.decls[v] = []nodeIndex{index}

			case varRedefined:
				// Redefinition
				if _, ok := d.namedReturn[v]; ok {
					// This is a redeclaration of a named return value.
					// Mark it so usage.go can filter it out (bare returns need
					// named returns to be accessible throughout the function).
					d.result.notMovable[index] = struct{}{}

					continue
				}

				// Append to the existing declaration list.
				// This allows tracking multiple declaration points for the same variable.
				d.result.decls[v] = append(d.result.decls[v], index)

			default:
				d.reportInternalError(n, "Unknown type of variable %s", v.Name())
			}
		}

	case *ast.DeclStmt:
		decl, ok := n.Decl.(*ast.GenDecl)
		if !ok || decl.Tok != token.VAR {
			break
		}

		for v, def := range d.allVars(allDeclared(decl)) {
			if def != varDefined || len(d.result.decls[v]) > 0 {
				d.reportInternalError(decl, "Re-definition of variable %s", v.Name())
				continue
			}

			d.result.decls[v] = []nodeIndex{c.Index()}
		}

	case *ast.File:
		return d.generated || !ast.IsGenerated(n)

	case *ast.FuncDecl:
		// Memory optimization: We only need the names of the current function (and nested function literals)
		clear(d.namedReturn)

	case *ast.FuncType:
		// Record named return values. These are implicitly available throughout the function body
		// and cannot be moved to tighter scopes (bare returns depend on them).
		results := n.Results
		if results == nil {
			break
		}

		for v, def := range d.allVars(allListed(results)) {
			if def != varDefined {
				d.reportInternalError(n, "Re-definition of return variable %s", v.Name())
				continue
			}

			d.namedReturn[v] = struct{}{}
		}
		// keep-sorted end
	}

	return true
}

type varType uint8

const (
	_ varType = iota
	varDefined
	varRedefined
)

func (p pass) allVars(ids iter.Seq[*ast.Ident]) iter.Seq2[*types.Var, varType] {
	return func(yield func(*types.Var, varType) bool) {
		for id := range ids {
			var (
				obj types.Object
				typ varType
			)

			if d, ok := p.pass.TypesInfo.Defs[id]; ok {
				if d == nil { // Symbolic variable in type switch (e.g., switch x := y.(type))
					continue
				}

				obj, typ = d, varDefined
			} else if u, ok := p.pass.TypesInfo.Uses[id]; ok {
				// Identifier not in Defs means it's a redefinition.
				obj, typ = u, varRedefined
			} else {
				p.reportInternalError(id, "unknown definition in declaration")
				continue
			}

			v, ok := obj.(*types.Var)
			if !ok {
				p.reportInternalError(id, "Non-var definition in declaration")
				continue
			}

			if !yield(v, typ) {
				return
			}
		}
	}
}
