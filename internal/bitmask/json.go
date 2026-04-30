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
	"encoding/json"
	"fmt"
	"math/bits"
	"strings"
	"unicode"
	"unsafe"
)

// MarshalJSON emits a JSON array of bit names in ascending bit position order;
// an empty mask is rendered as "[]".
func MarshalJSON(mask uint64, names []string) ([]byte, error) {
	if mask == 0 {
		return []byte("[]"), nil
	}

	if len(names) < bits.Len64(mask) {
		return nil, EncodingError(mask)
	}

	const (
		prefix    = `["`
		separator = `","`
		suffix    = `"]`
	)

	size := len(prefix) + len(suffix)
	size += calcSize(mask, names, len(separator))

	buf := make([]byte, 0, size)
	buf = append(buf, prefix...)
	buf = appendNames(buf, mask, names, separator)
	buf = append(buf, suffix...)

	return buf, nil
}

// UnmarshalJSON parses a JSON encoding of a bitmask into a uint64. It accepts:
//   - a bool (when trueValue != 0): "true" yields trueValue; "false" and
//     "[]" yield 0. Passing trueValue == 0 disables bool input.
//   - a string: parsed via [UnmarshalText] / [unmarshalTextString], so the
//     same comma- or whitespace-separated grammar applies as for plain text.
//   - an array of strings: each element is resolved via parseName and
//     accumulated, with a token resolving to 0 resetting the accumulator
//     (the same zero-reset semantic as [UnmarshalText]).
func UnmarshalJSON(
	data []byte,
	parseName func(string) (uint64, bool),
	trueValue uint64,
) (uint64, error) {
	data = bytes.TrimSpace(data)

	if len(data) == 0 {
		return 0, fmt.Errorf("cannot unmarshal %s", data)
	}

	switch data[0] {
	case '"':
		return unmarshalJSONString(data, parseName)

	case '[':
		return unmarshalJSONArray(data, parseName)

	case 't':
		if string(data) == "true" {
			if trueValue == 0 {
				return 0, DecodingError(data)
			}

			return trueValue, nil
		}

	case 'f':
		if string(data) == "false" {
			if trueValue == 0 {
				return 0, DecodingError(data)
			}

			return 0, nil
		}

	case 'n':
		if string(data) == "null" {
			return 0, nil
		}
	}

	return 0, fmt.Errorf("cannot unmarshal %s", data)
}

// unmarshalJSONString resolves one JSON element to its bit value. A quoted
// string with no escape is parsed straight from its inner bytes (no decode, no
// allocation); an escaped string is decoded via encoding/json so sequences like
// \t and \uXXXX resolve. A non-string element is rejected.
func unmarshalJSONString(
	data []byte,
	parseName func(string) (uint64, bool),
) (uint64, error) {
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		if text := data[1 : len(data)-1]; bytes.IndexByte(text, '\\') < 0 {
			return UnmarshalText(text, parseName)
		}
	}

	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return 0, fmt.Errorf("expected string: %w", err)
	}

	return unmarshalTextString(text, parseName)
}

// unmarshalTextString is the string-input counterpart of [UnmarshalText],
// taking a string and a string-based parseName so the caller need not convert
// to []byte. Behavior, semantics, and error type are identical.
func unmarshalTextString(text string, parseName func(string) (uint64, bool)) (uint64, error) {
	var v uint64

	for word := range strings.FieldsFuncSeq(text, func(r rune) bool { return r == ',' || unicode.IsSpace(r) }) {
		mask, ok := parseName(word)
		if !ok {
			return 0, DecodingError(word)
		}

		v = accumulate(v, mask)
	}

	return v, nil
}

// unmarshalJSONArray decodes a JSON array of strings from its raw bytes: each
// element is resolved by name and accumulated, with a token resolving to 0
// resetting the accumulator.
func unmarshalJSONArray(
	data []byte,
	parseName func(string) (uint64, bool),
) (uint64, error) {
	if string(data) == "[]" {
		return 0, nil
	}

	var elems []json.RawMessage
	if err := json.Unmarshal(data, &elems); err != nil {
		return 0, err
	}

	var v uint64

	for _, elem := range elems {
		mask, err := unmarshalJSONArrayString(elem, parseName)
		if err != nil {
			return 0, err
		}

		v = accumulate(v, mask)
	}

	return v, nil
}

// unmarshalJSONArrayString resolves one JSON element to its bit value. A quoted
// string with no escape is parsed straight from its inner bytes (no decode, no
// allocation); an escaped string is decoded via encoding/json so sequences like
// \t and \uXXXX resolve. A non-string element is rejected.
func unmarshalJSONArrayString(
	data []byte,
	parseName func(string) (uint64, bool),
) (uint64, error) {
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		if word := data[1 : len(data)-1]; bytes.IndexByte(word, '\\') < 0 {
			// #nosec G103 -- safe, parseName doesn't keep a reference.
			if mask, ok := parseName(unsafe.String(unsafe.SliceData(word), len(word))); ok {
				return mask, nil
			}

			return 0, DecodingError(word)
		}
	}

	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return 0, fmt.Errorf("expected string: %w", err)
	}

	if mask, ok := parseName(text); ok {
		return mask, nil
	}

	return 0, DecodingError(text)
}
