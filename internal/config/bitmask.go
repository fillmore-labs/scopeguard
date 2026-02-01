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

// BitMask is a generic type that represents a bitmask for managing binary flags.
type BitMask[T ~uint8 | ~uint16 | ~uint32 | ~uint64] struct { // constraints.Integer would be fine, but it lives in golang.org/x/exp
	value T
}

// NewBitMask creates a new typed [BitMask] instance with the specified flags enabled.
func NewBitMask[T ~uint8 | ~uint16 | ~uint32 | ~uint64](flags ...T) BitMask[T] {
	var b BitMask[T]
	for _, flag := range flags {
		b.Enable(flag)
	}

	return b
}

// Set adjusts the bitmask by enabling or disabling the specified option.
func (b *BitMask[T]) Set(flag T, value bool) {
	if value {
		b.Enable(flag)
	} else {
		b.Disable(flag)
	}
}

// Enable sets the given flag in the current bitmask, enabling the specified option.
func (b *BitMask[T]) Enable(flag T) {
	b.value |= flag
}

// Disable removes the specified flag from the current bitmask, disabling the associated option.
func (b *BitMask[T]) Disable(flag T) {
	b.value &^= flag
}

// Enabled checks if the specified option is enabled in the current bitmask.
func (b BitMask[T]) Enabled(flag T) bool {
	return b.value&flag != 0
}
