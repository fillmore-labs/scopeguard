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

func set[T interface{ ~uint8 | ~uint16 | ~uint32 }](t *T, flag T, value bool) {
	if value {
		*t |= flag
	} else {
		*t &^= flag
	}
}

// toString converts a bitflag into a human-readable string representation.
// The `names` slice should contain entries indexed by bit position (0, 1, 2, etc.)
// corresponding to the bit flags at positions 0, 1, 2.
// Multiple set flags are joined with ", " in descending bit position order.
// Unknown flags (positions beyond the names slice) are represented as "Unknown(N)" where N is the bit position.
func toString[T interface{ ~uint8 | ~uint16 | ~uint32 }](t T, names []string, none string) string {
	if t == 0 {
		return none
	}

	nameCount := len(names)

	parts := make([]string, 0, bits.OnesCount32(uint32(t)))
	for remaining, pos := t, 0; remaining != 0; remaining &^= T(1) << pos {
		pos = bits.Len32(uint32(remaining)) - 1 // Highest bit position

		var name string
		if pos < nameCount {
			name = names[pos]
		} else {
			name = fmt.Sprintf("Unknown(%d)", pos)
		}

		parts = append(parts, name)
	}

	return strings.Join(parts, ", ")
}
