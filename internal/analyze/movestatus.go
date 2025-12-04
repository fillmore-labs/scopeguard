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

// MoveStatus indicates whether a declaration can be moved and why.
type MoveStatus uint8

//go:generate go tool stringer -type MoveStatus -linecomment
const (
	// MoveAllowed indicates the declaration can be safely moved.
	MoveAllowed MoveStatus = iota // mov
	// MoveBlockedInitConflict indicates the move is blocked by an Init field conflict.
	MoveBlockedInitConflict // ini
	// MoveBlockedTypeIncompatible indicates the move is blocked by type incompatibility.
	MoveBlockedTypeIncompatible // typ
	// MoveBlockedGenerated indicates the move is blocked because the file is generated.
	MoveBlockedGenerated // gen
	// MoveBlockedDeclared indicates the move is blocked by a declaration in the target scope.
	MoveBlockedDeclared // dec
	// MoveBlockedShadowed indicates the move is blocked due to shadowing of variables used in the declaration.
	MoveBlockedShadowed // shw
	// MoveBlockedTypeChange indicates the move is blocked because it would change the type of a variable.
	MoveBlockedTypeChange // tch
	// MoveBlockedStatements indicates the move is blocked because of intervening statements.
	MoveBlockedStatements // xst
)

// Movable indicates the declaration would be moved.
func (m MoveStatus) Movable() bool { return m == MoveAllowed }
