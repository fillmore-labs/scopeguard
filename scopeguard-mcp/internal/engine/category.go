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

// Package category defines the diagnostic classification types used throughout
// the ScopeGuard analyzer pipeline.
package engine

import (
	"fmt"
	"strconv"

	"fillmore-labs.com/scopeguard/internal/category"
	"fillmore-labs.com/scopeguard/internal/config"
)

// Info bundles the classification metadata for a single diagnostic category.
type Info struct {
	Reason string
	Issue  Issue
	Safety config.Safety
}

// InfoOf returns information about the given category.
func InfoOf(category string) *Info {
	info, ok := infoByCategory[category]
	if !ok {
		return &Info{Issue: IssueUnknown, Reason: "Internal error: unknown category " + strconv.Quote(category), Safety: config.Unknown}
	}

	return info
}

var reasons = [...]string{
	category.MoveAllowed:                 "",
	category.MoveBlockedInitConflict:     "Conflicting moves to initialization statement: combine them or select only one",
	category.MoveBlockedTypeIncompatible: "Type information lost at new declaration site: add an explicit type annotation or split the declaration",
	category.MoveBlockedGenerated:        "Do not edit generated files",
	category.MoveBlockedDeclared:         "Redeclaration in target scope: same name already declared there; rename one before applying",
	category.MoveBlockedShadowed:         "Shadowed identifier after move: initializer would bind to a different variable; introduce a temporary or skip",
	category.MoveBlockedTypeChange:       "Inferred type changes for later reassignment: add an explicit type annotation or keep the original declaration in place",
	category.MoveSideEffects:             "Moving may change evaluation order relative to side effects",
}

// infoByCategory is an internal map pointing to category info by category.
var infoByCategory = func() map[string]*Info {
	infoByCategory := make(map[string]*Info, category.MaxMoveStatus+2)
	for i := range category.MaxMoveStatus {
		info := &Info{
			Reason: reasons[i],
			Issue:  IssueScope,
			Safety: i.Safety(),
		}

		if info.Safety != config.Safe && info.Reason == "" {
			panic(fmt.Sprintf("Missing reason for unsafe MoveStatus %v", i))
		}

		cat := i.String()
		infoByCategory[cat] = info
	}

	infoByCategory[category.Shadowed] = &Info{
		Reason: "", Issue: IssueShadow, Safety: config.Safe,
	}

	infoByCategory[category.NestedAssignment] = &Info{
		Reason: "Nested reassignment in closure: requires manual restructuring", Issue: IssueNestedAssignment, Safety: config.Safe,
	}

	return infoByCategory
}()
