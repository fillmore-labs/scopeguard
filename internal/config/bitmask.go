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
	"slices"
	"strings"
)

type bitmask interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64
}

func set[B bitmask](v *B, b B, value bool) {
	if value {
		*v |= b
	} else {
		*v &^= b
	}
}

func enabled[B bitmask](v, b B) bool {
	return v&b == b
}

// toStrings converts a [bitmask] into a human-readable string representation.
// The `names` slice should contain entries indexed by bit position (0, 1, 2, etc.)
// corresponding to the bit flags at positions 0, 1, 2.
// Unknown flags (positions beyond the names slice) are represented as "Unknown(N)" where N is the bit position.
func toStrings[B bitmask](b B, names, none []string) []string {
	if b == 0 {
		return none
	}

	parts := make([]string, 0, bits.OnesCount64(uint64(b)))
	for remaining, pos := b, 0; remaining != 0; remaining &^= B(1) << pos {
		pos = bits.Len64(uint64(remaining)) - 1 // Highest bit position

		var name string
		if pos < len(names) {
			name = names[pos]
		} else {
			name = fmt.Sprintf("Unknown(%d)", pos)
		}

		parts = append(parts, name)
	}

	slices.Reverse(parts)

	return parts
}

func toString[B bitmask](b B, names []string) string {
	return strings.Join(toStrings(b, names, []string{"none"}), ", ")
}
