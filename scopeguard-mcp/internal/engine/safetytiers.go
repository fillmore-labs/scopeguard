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

package engine

import "fillmore-labs.com/scopeguard/internal/config"

// SafetyTiers wraps [config.Safety] for JSON marshaling as a list of
// tier names. It carries the same orientation as the underlying filter:
// each set bit represents a tier that is included.
//
// In tool input structs, use *SafetyTiers so a nil pointer signals
// "default: include all tiers"; a non-nil value carries an explicit
// selection (which may be empty, meaning "include no tiers").
//
//go:generate go tool bitmask -type SafetyTiers -json
type SafetyTiers config.Safety

const (
	// Safe means the fix can be applied without risk.
	Safe = SafetyTiers(config.Safe) // safe

	// Unsafe means the fix is structurally valid but may reorder side effects.
	Unsafe = SafetyTiers(config.Unsafe) // unsafe

	// Breaking means the fix could break compilation; requires manual review.
	Breaking = SafetyTiers(config.Breaking) // breaking

	// None suppresses everything.
	None = SafetyTiers(config.Nothing) // none
)

// Filter returns the underlying [config.Safety], or [config.All] when the receiver is nil.
// Call sites use this to map "field omitted" to the default "include every tier".
func (s *SafetyTiers) Filter() config.Safety {
	if s == nil {
		return config.All
	}

	return config.Safety(*s)
}

// ValidTiers returns a slice of valid safety tier names.
func ValidTiers() []string {
	return _SafetyTiersNames[:]
}
