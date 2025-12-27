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

// MoveStatus indicates whether a declaration can be moved and why.
type MoveStatus uint8

//go:generate go tool stringer -type MoveStatus -linecomment
const (
	// MoveAllowed indicates the declaration can be safely moved.
	MoveAllowed MoveStatus = iota // mov

	// MoveBlockedInitConflict indicates the move is blocked by an Init field conflict.
	// This happens when multiple declarations target the same init field and cannot be combined.
	// Users can resolve this by enabling CombineDeclarations or manually combining them.
	MoveBlockedInitConflict // ini

	// MoveAbsorbed indicates the declaration is merged into another move.
	// This status is informational and does not represent a blocked state.
	// It occurs when CombineDeclarations is enabled.
	MoveAbsorbed // abs

	// MoveBlockedTypeIncompatible indicates the move is blocked by type incompatibility.
	// Moving the declaration would cause subsequent code to infer a different type,
	// potentially breaking compilation or changing semantics.
	MoveBlockedTypeIncompatible // typ

	// MoveBlockedGenerated indicates the move is blocked because the file is generated.
	// We do not generate fixes for generated files.
	MoveBlockedGenerated // gen

	// MoveBlockedDeclared indicates the move is blocked by an existing declaration in the target scope.
	// Moving the variable would cause a redeclaration error.
	MoveBlockedDeclared // dec

	// MoveBlockedShadowed indicates the move is blocked due to shadowing of variables used in the declaration.
	// Moving the declaration would change which variable identifiers refer to.
	MoveBlockedShadowed // shw

	// MoveBlockedTypeChange indicates the move is blocked because it would change the type of a variable.
	// This ensures that type inference remains consistent.
	MoveBlockedTypeChange // tch

	// MoveBlockedStatements indicates the move is blocked because of intervening statements.
	// This only applies in conservative mode, where any potential side effect blocks a move.
	MoveBlockedStatements // xst
)

// Movable indicates the declaration could be moved.
func (i MoveStatus) Movable() bool { return i == MoveAllowed }
