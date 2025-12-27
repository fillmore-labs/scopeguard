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

package scope

import (
	"go/ast"

	"golang.org/x/tools/go/ast/astutil"
)

// Name returns a human-readable name for the scope type.
func Name(node ast.Node) string {
	switch node.(type) {
	// keep-sorted start newline_separated=yes
	case *ast.BlockStmt:
		return "block"

	case *ast.CaseClause:
		return "case"

	case *ast.CommClause:
		return "select case"

	case *ast.File:
		return "file"

	case *ast.ForStmt:
		return "for"

	case *ast.FuncType:
		return "function"

	case *ast.IfStmt:
		return "if"

	case *ast.RangeStmt:
		return "range"

	case *ast.SwitchStmt:
		return "switch"

	case *ast.TypeSwitchStmt:
		return "type switch"

	case nil:
		return "<nil>"

	default:
		return astutil.NodeDescription(node)
		// keep-sorted end
	}
}
