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

package astutil

import (
	"go/ast"

	"golang.org/x/tools/go/ast/inspector"
)

// NodeIndex is the index from [inspector.Cursor], increases monotonically throughout the traversal.
type NodeIndex int32

// InvalidNode represents an invalid node index.
const InvalidNode NodeIndex = -1

// NodeIndexOf returns the [NodeIndex] for the current [inspector.Cursor] position.
func NodeIndexOf(c inspector.Cursor) NodeIndex {
	return NodeIndex(c.Index())
}

// Valid checks if this index is valid.
func (n NodeIndex) Valid() bool {
	return n != InvalidNode
}

// Cursor returns the [inspector.Cursor] corresponding to this index.
func (n NodeIndex) Cursor(in *inspector.Inspector) inspector.Cursor {
	return in.At(int32(n))
}

// Node returns the [ast.Node] corresponding to this index.
func (n NodeIndex) Node(in *inspector.Inspector) ast.Node {
	return in.At(int32(n)).Node()
}
