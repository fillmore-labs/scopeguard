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

package target

import (
	"go/ast"

	"fillmore-labs.com/scopeguard/internal/astutil"
)

// MoveTarget represents a declaration that can be moved to a tighter scope.
type MoveTarget struct {
	MovableDecl                 // The declaration to move
	TargetNode    ast.Node      // The node with the target scope (e.g., *[ast.IfStmt], *[ast.BlockStmt])
	AbsorbedDecls []MovableDecl // Additional declarations merged into this one
	Status        MoveStatus    // Status indicating if the move is safe or why it isn't
}

// MovableDecl represents a declaration that can be moved to another scope in the code analysis process.
type MovableDecl struct {
	Decl   astutil.NodeIndex // Inspector index of the declaration statement to move
	Unused []string          // Unused identifiers in this declaration
}

// MoveStatus indicates if a move is safe or why it isn't.
// Implementations report specific reasons that prevent moving
// a declaration (e.g., variable shadowing, scope conflicts).
type MoveStatus interface {
	Movable() bool
	String() string
}
