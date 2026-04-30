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

import (
	"fmt"
	"math/bits"
	"strings"
)

// Safety indicates whether applying a fix for a diagnostic [Category] is safe,
// may reorder side effects (unsafe), or would likely break compilation (breaking).
type Safety uint8

const (
	// Safe - fix can be applied without risk.
	Safe Safety = 1 << iota

	// Unsafe - fix is structurally valid but may reorder side effects.
	Unsafe

	// Breaking - fix could break compilation; requires manual review.
	Breaking

	// Unknown - unrecognized category; should not appear in practice.
	Unknown Safety = 0
)

// safetyNames lists the recognized safety tier names.
var safetyNames = [...]string{
	"unknown",
	"safe",
	"unsafe",
	"breaking",
}

func (s Safety) String() string {
	l := bits.Len8(uint8(s))
	if l >= len(safetyNames) {
		return fmt.Sprintf("Safety(%d)", l)
	}

	return safetyNames[l]
}

// ValidSafetyNames returns a slice of valid safety tier names, excluding the "unknown" tier.
func ValidSafetyNames() []string {
	return safetyNames[1:]
}

// MarshalText implements [encoding.TextMarshaler].
func (s Safety) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// UnmarshalText implements [encoding.TextUnmarshaler].
func (s *Safety) UnmarshalText(text []byte) error {
	v := string(text)
	for i, t := range safetyNames[1:] {
		if v == t {
			*s = Safety(1 << i)
			return nil
		}
	}

	return fmt.Errorf("unknown safety: %q (valid values: %s)", v, strings.Join(ValidSafetyNames(), ", "))
}

// SafetyFilter is a bitmask for filtering on the safety level of a diagnostic.
type SafetyFilter uint8

const (
	// FilterSafe represents the 'Safe' category, indicating changes that can be applied without risk.
	FilterSafe = SafetyFilter(Safe)

	// FilterUnsafe represents the 'Unsafe' category, indicating changes that may reorder side effects.
	FilterUnsafe = SafetyFilter(Unsafe)

	// FilterBreaking represents the 'Breaking' category, indicating changes that could break compilation.
	FilterBreaking = SafetyFilter(Breaking)

	// FilterAll is the default, filtering nothing.
	FilterAll = FilterSafe | FilterUnsafe | FilterBreaking

	// FilterNothing suppresses everything.
	FilterNothing = SafetyFilter(Unknown)
)

// Set adjusts the bitmask by enabling or disabling the specified option.
func (s *SafetyFilter) Set(filter SafetyFilter, value bool) { set(s, filter, value) }

// Enabled checks if the specified option is enabled in the current bitmask.
func (s SafetyFilter) Enabled(filter SafetyFilter) bool { return enabled(s, filter) }

// Default checks if the SafetyTiers value is set to the default state.
func (s SafetyFilter) Default() bool {
	return s == FilterAll
}

// Allowed determines if a specific safety tier is enabled within the filter.
func (s SafetyFilter) Allowed(c Safety) bool {
	return s.Enabled(SafetyFilter(c))
}

// SetSafety adjusts the bitmask by enabling or disabling the specified safety tier.
func (s *SafetyFilter) SetSafety(c Safety, value bool) {
	s.Set(SafetyFilter(c), value)
}

// Tiers convert a filter into a list of tier names.
func (s SafetyFilter) Tiers() []string {
	return toStrings(s, safetyNames[1:], []string{})
}

// Filters is a pair of [SafetyFilter] values used to filter diagnostics and suggested fix generation based on [Safety] levels.
type Filters [2]SafetyFilter

// NewFilters creates a [Filters] instance with the specified diagnostic and fix generation [SafetyFilter] settings.
func NewFilters(diagnostic, fix SafetyFilter) Filters {
	return Filters{diagnostic, fix}
}

// Diagnostic returns the first SafetyFilter in the Filters array, used to filter diagnostics.
func (f Filters) Diagnostic() SafetyFilter {
	return f[0]
}

// Fix returns the second element of the Filters array, representing the safety filter for suggested fixes.
func (f Filters) Fix() SafetyFilter {
	return f[1]
}
