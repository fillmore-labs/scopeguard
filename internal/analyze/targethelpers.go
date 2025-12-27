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

package analyze

import (
	"go/ast"
	"go/token"
	"iter"
	"slices"

	"golang.org/x/tools/go/ast/inspector"
)

// nextLabel returns the closest label position in labels after pos, or token.NoPos if not found.
func nextLabel(labels []token.Pos, pos token.Pos) token.Pos {
	l := len(labels)
	if l == 0 {
		return token.NoPos
	}

	if i, _ := slices.BinarySearch(labels, pos); i < l {
		return labels[i]
	}

	return token.NoPos
}

// declInfo extracts assigned identifiers and whether the move is restricted to block statements only.
func declInfo(declNode ast.Node, cf CurrentFile, maxLines int) (identifiers iter.Seq[*ast.Ident], onlyBlock bool) {
	switch n := declNode.(type) {
	case *ast.AssignStmt:
		// Short declarations can go to init fields if they're small enough
		return allAssigned(n), maxLines > 0 && cf.Lines(declNode) > maxLines

	case *ast.DeclStmt:
		// var declarations can only go to block statements (not init fields)
		return allDeclared(n), true

	default:
		// Unsupported declaration type
		return nil, false
	}
}

// sortedLabels collects all labeled statement positions in the function body.
//
// These positions are used to prevent declaration moves across labels,
// which could change program semantics.
//
// Returns nil if no labels are found, otherwise returns sorted positions.
func sortedLabels(body inspector.Cursor) []token.Pos {
	var labels []token.Pos
	for l := range body.Preorder((*ast.LabeledStmt)(nil)) {
		labels = append(labels, l.Node().Pos())
	}

	if len(labels) == 0 {
		return nil
	}

	// Sort positions to enable binary search during candidate analysis.
	// While Preorder traverses in depth-first order (which typically matches source order),
	// explicit sorting ensures correctness and is negligible overhead.
	slices.Sort(labels)

	return labels
}

// enclosingInterval determines the execution interval [start, end) between the declaration
// and the target node and finds the innermost parent that spans this interval.
func enclosingInterval(declCursor inspector.Cursor, targetNode ast.Node) (parent inspector.Cursor, start, end token.Pos) {
	start, end = declCursor.Node().End(), targetNode.Pos()

	parent = declCursor.Parent()
	for !extendsUntil(parent, end) {
		parent = parent.Parent()
	}

	return parent, start, end
}

// extendsUntil returns true when the current parent node extends until end.
func extendsUntil(parent inspector.Cursor, end token.Pos) bool {
	parentNode := parent.Node()

	return parentNode == nil || parentNode.End() >= end
}

// initField determines whether the targetNode is an initialization field in a control structure.
func initField(targetNode ast.Node) bool {
	switch targetNode.(type) {
	case *ast.IfStmt,
		*ast.ForStmt,
		*ast.SwitchStmt,
		*ast.TypeSwitchStmt:
		return true

	default:
		return false
	}
}

// usedAndTypeChange tests whether a type change in a declaration would affect semantics.
func usedAndTypeChange(flags UsageFlags, conservative bool) bool {
	// Check if both Used and TypeChange flags are set
	usedAndTypeChange := flags&UsageUsedAndTypeChange == UsageUsedAndTypeChange

	// Block in conservative mode or when untyped nil is involved
	return usedAndTypeChange && (conservative || flags&UsageUntypedNil != 0)
}
