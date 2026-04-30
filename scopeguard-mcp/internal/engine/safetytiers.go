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

import (
	"encoding/json"

	"fillmore-labs.com/scopeguard/internal/config"
)

// SafetyTiers wraps [config.SafetyFilter] for JSON marshaling as a list of
// tier names. It carries the same orientation as the underlying filter:
// each set bit represents a tier that is included.
//
// In tool input structs, use *SafetyTiers so a nil pointer signals
// "default: include all tiers"; a non-nil value carries an explicit
// selection (which may be empty, meaning "include no tiers").
type SafetyTiers config.SafetyFilter

// Filter returns the underlying [config.SafetyFilter], or [config.FilterAll] when
// the receiver is nil. Call sites use this to map "field omitted" to the default
// "include every tier".
func (s *SafetyTiers) Filter() config.SafetyFilter {
	if s == nil {
		return config.FilterAll
	}

	return config.SafetyFilter(*s)
}

// MarshalJSON implements [json.Marshaler]. It emits the included tiers as a
// JSON array of names; an empty filter marshals to "[]" so the round trip
// through [SafetyTiers.UnmarshalJSON] is stable.
func (s SafetyTiers) MarshalJSON() ([]byte, error) {
	f := config.SafetyFilter(s)

	return json.Marshal(f.Tiers())
}

// UnmarshalJSON implements [json.Unmarshaler]. The input must be a JSON array
// of tier names; each name sets the corresponding bit in the resulting
// filter. An empty array yields [config.FilterNothing] (no tiers included).
func (s *SafetyTiers) UnmarshalJSON(data []byte) error {
	var tiers []config.Safety
	if err := json.Unmarshal(data, &tiers); err != nil {
		return err
	}

	if tiers == nil {
		*s = SafetyTiers(config.FilterAll)
		return nil
	}

	var f config.SafetyFilter
	for _, c := range tiers {
		f.SetSafety(c, true)
	}

	*s = SafetyTiers(f)

	return nil
}
