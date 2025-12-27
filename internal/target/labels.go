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

package target

import (
	"go/ast"
	"go/token"
	"slices"

	"golang.org/x/tools/go/ast/inspector"
)

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
