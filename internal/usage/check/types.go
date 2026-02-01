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

package check

import (
	"go/ast"
	"go/token"
	"go/types"

	"fillmore-labs.com/scopeguard/internal/astutil"
)

// ShadowUse contains information about a variable use after previously shadowed.
type ShadowUse struct {
	Var       *types.Var
	ShadowPos token.Pos
	Use       astutil.NodeIndex
}

// NestedAssign contains information about a nested variable assign.
type NestedAssign struct {
	Ident *ast.Ident
	Asgn  astutil.NodeIndex
}
