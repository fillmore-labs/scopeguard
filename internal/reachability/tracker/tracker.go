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

package tracker

import (
	"go/ast"
	"go/types"
)

// Tracker provides methods for analyzing functions.
type Tracker struct {
	info *types.Info // Type information for identifying functions that can't return
}

// CantReturn determines if the given function call expression represents a function that cannot return.
func (t *Tracker) CantReturn(n *ast.CallExpr) bool {
	return CantReturn(t.info, n)
}

// New creates and returns a new Tracker.
func New(info *types.Info) Tracker {
	return Tracker{
		info: info,
	}
}
