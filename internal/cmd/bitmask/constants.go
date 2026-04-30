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

package main

import (
	"cmp"
	"errors"
	"fmt"
	"go/token"
	"math/bits"
	"slices"
)

// typeConstants buckets the line-commented constants of a single type, keyed while parsing.
type typeConstants struct {
	// single maps a bit position to its const.
	single map[int]constInfo
	// combos collects multi-bit combinations (later validated into aliases by createSpecs).
	combos []constInfo
}

// constInfo is one line-commented constant parsed from source: its identifier,
// serialized name (the line comment), value, and source position.
type constInfo struct {
	Const string // const identifier (e.g. "BitmaskAll")
	Name  string // serialized name from the line comment (e.g. "all")
	value uint64
	pos   token.Pos // source position, used to order aliases
}

func (c constInfo) compare(other constInfo) int {
	return cmp.Compare(c.pos, other.pos)
}

// classify classifies one parsed constant into its bucket:
// the single-bit table (by position), or the multi-bit combos slice.
func (c *typeConstants) classify(cd constInfo) error {
	if bits.OnesCount64(cd.value) != 1 {
		// Same-value combos are rejected later in buildAliases; sortCombos orders
		// c.combos into source-declaration order beforehand so the error names the
		// earlier-declared const first.
		c.combos = append(c.combos, cd)
		return nil
	}

	// Single-bit constant.
	bitPos := bits.TrailingZeros64(cd.value)
	if dup, exists := c.single[bitPos]; exists {
		// Two single-bit consts claim the same bit position.
		return fmt.Errorf("duplicate bit %d: %s and %s", bitPos, dup.Const, cd.Const)
	}

	c.single[bitPos] = cd

	return nil
}

// buildBits returns one [constInfo] per single-bit constant, indexed by bit
// position. Positions must be contiguous from 0.
func (c *typeConstants) buildBits() ([]constInfo, error) {
	numBits := len(c.single)
	if numBits == 0 {
		return nil, errors.New("no single-bit constant with a line comment found")
	}

	names := make([]constInfo, numBits)
	for bitPos := range numBits {
		cd, ok := c.single[bitPos]
		if !ok {
			return nil, fmt.Errorf("missing constant with a line comment for bit position %d (positions must be contiguous from 0)", bitPos)
		}

		names[bitPos] = cd
	}

	return names, nil
}

// buildAliases returns multi-bit alias constants, rejecting two aliases that
// share the same value (AppendText would be ambiguous about which name to
// render) and aliases that set a bit beyond the defined single-bit constants
// (those would round-trip as text but fail to marshal as JSON).
//
// It must run after buildBits, which validates that the single bits occupy
// positions 0..len(single)-1 contiguously; that lets bits.Len64 alone decide
// whether every bit of an alias is named.
//
// The output preserves the order of c.combos; the caller must pre-sort it
// into source-declaration order.
func (c *typeConstants) buildAliases() ([]constInfo, error) {
	numBits := len(c.single)

	seenValue := make(map[uint64]string, len(c.combos))
	for _, combo := range c.combos {
		if other, ok := seenValue[combo.value]; ok {
			return nil, fmt.Errorf("aliases %s and %s share the same value", other, combo.Const)
		}

		if bits.Len64(combo.value) > numBits {
			return nil, fmt.Errorf("alias %s (0x%x) sets a bit beyond the %d defined single-bit constant(s)",
				combo.Const, combo.value, numBits)
		}

		seenValue[combo.value] = combo.Const
	}

	c.combos = slices.Clip(c.combos)

	return c.combos, nil
}

// sortCombos sorts multi-bit alias combos into source-declaration order.
func (c *typeConstants) sortCombos() {
	slices.SortFunc(c.combos, constInfo.compare)
}
