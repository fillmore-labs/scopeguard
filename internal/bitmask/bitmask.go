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

package bitmask

import (
	"bytes"
	"math/bits"
	"slices"
	"strconv"
	"unicode"
	"unsafe"
)

// EncodingError is returned by [AppendText] when mask has a set bit beyond the
// supplied names.
//
// Its underlying uint64 is the offending mask value.
type EncodingError uint64

// Error reports the offending value in hexadecimal.
func (e EncodingError) Error() string {
	return "invalid value 0x" + strconv.FormatUint(uint64(e), 16)
}

// DecodingError is returned when [UnmarshalText], or [UnmarshalJSON] encounter
// a token that parseName does not recognize.
//
// Its underlying string is the offending token.
type DecodingError string

// Error reports the offending token, quoted for safety against unusual characters.
func (e DecodingError) Error() string {
	return "unknown " + strconv.Quote(string(e))
}

// FormatText renders mask as a string: the names of the set bits joined with
// ", " in ascending bit position order, matching [AppendText].
//
// names[i] is the name of the bit at position i.
//
// It returns an [EncodingError] when a set bit lacks a name.
func FormatText(mask uint64, names []string) (string, error) {
	if bits.OnesCount64(mask) == 1 {
		if pos := bits.TrailingZeros64(mask); pos < len(names) {
			return names[pos], nil
		}
	}

	txt, err := AppendText(nil, mask, names)

	// #nosec G103 -- safe, AppendText allocates a new slice.
	return unsafe.String(unsafe.SliceData(txt), len(txt)), err
}

// AppendText converts mask into a human-readable string representation and
// appends it to buf. Multiple set flags are joined with ", " in ascending bit
// position order.
//
// names[i] is the name of the bit at position i.
//
// It returns an [EncodingError] when a set bit lacks a name.
func AppendText(buf []byte, mask uint64, names []string) ([]byte, error) {
	if mask == 0 {
		return buf, nil
	}

	if len(names) < bits.Len64(mask) {
		return nil, EncodingError(mask)
	}

	const separator = ", "

	// Pre-size buf to the exact length.
	size := calcSize(mask, names, len(separator))
	buf = slices.Grow(buf, size)

	return appendNames(buf, mask, names, separator), nil
}

// calcSize calculates the required size.
func calcSize(mask uint64, names []string, sepLen int) int {
	size := (bits.OnesCount64(mask) - 1) * sepLen
	for remaining := mask; remaining != 0; remaining &= remaining - 1 {
		pos := bits.TrailingZeros64(remaining)
		size += len(names[pos])
	}

	return size
}

// appendNames appends mask's bit names to buf, joined by sep.
func appendNames(buf []byte, mask uint64, names []string, sep string) []byte {
	first := true
	for remaining := mask; remaining != 0; remaining &= remaining - 1 {
		if !first {
			buf = append(buf, sep...)
		}
		first = false

		pos := bits.TrailingZeros64(remaining)
		buf = append(buf, names[pos]...)
	}

	return buf
}

// UnmarshalText splits text on commas or whitespace, resolves each token via
// parseName, and accumulates the matching bits.
//
// parseName reports (value, true) for a recognized token and (_, false) for
// an unknown one. A recognized token whose value is 0 (the zero-valued name)
// resets the accumulator, so "all,off" yields 0 rather than the bits of "all";
// this preserves the convention that the zero name clears any prior bits.
//
// On the first unknown token, UnmarshalText returns a [DecodingError] carrying
// that token. Callers typically wrap the error with their type name.
func UnmarshalText(text []byte, parseName func(string) (uint64, bool)) (uint64, error) {
	var v uint64

	for word := range bytes.FieldsFuncSeq(text, func(r rune) bool { return r == ',' || unicode.IsSpace(r) }) {
		// #nosec G103 -- safe, parseName doesn't keep a reference.
		mask, ok := parseName(unsafe.String(unsafe.SliceData(word), len(word)))
		if !ok {
			return 0, DecodingError(word)
		}

		v = accumulate(v, mask)
	}

	return v, nil
}

// accumulate folds bit into v; the zero name resets the accumulator.
func accumulate(v, mask uint64) uint64 {
	if mask == 0 {
		return 0
	}

	return v | mask
}
