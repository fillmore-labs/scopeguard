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

package config

// Safety indicates whether applying a fix for a diagnostic [Category] is safe,
// may reorder side effects (unsafe), or would likely break compilation (breaking).
//
// It is used as a bitmask for filtering on the safety level of a diagnostic.
//
//go:generate go tool bitmask -boolflag -type Safety
type Safety uint8

const (
	// Safe means the fix can be applied without risk.
	Safe Safety = 1 << iota // safe

	// Unsafe means the fix is structurally valid but may reorder side effects.
	Unsafe // unsafe

	// Breaking means the fix could break compilation; requires manual review.
	Breaking // breaking

	// All is the default filter, filtering nothing.
	All Safety = 1<<iota - 1 // all

	// Nothing suppresses everything.
	Nothing Safety = 0 // none
)

// ValidSafeties returns a slice of valid single-bit safety names.
func ValidSafeties() []string {
	return _SafetyNames[:]
}

// Default checks if the SafetyTiers value is set to the default state.
func (s Safety) Default() bool {
	return s&All == All
}

// Filters is a pair of [SafetyFilter] values used to filter diagnostics and suggested fix generation based on [Safety] levels.
type Filters [2]Safety

// NewFilters creates a [Filters] instance with the specified diagnostic and fix generation [SafetyFilter] settings.
func NewFilters(diagnostic, fix Safety) Filters {
	return Filters{diagnostic, fix}
}

// Diagnostic returns the first SafetyFilter in the Filters array, used to filter diagnostics.
func (f Filters) Diagnostic() Safety {
	return f[0]
}

// Fix returns the second element of the Filters array, representing the safety filter for suggested fixes.
func (f Filters) Fix() Safety {
	return f[1]
}
