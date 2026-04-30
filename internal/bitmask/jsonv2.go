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

//go:build go1.27 && goexperiment.jsonv2

package bitmask

import (
	"encoding/json/jsontext"
	"fmt"
	"math/bits"
)

// MarshalJSONTo emits a JSON array of bit names in ascending bit position order;
// aliases are not used on output.
func MarshalJSONTo(enc *jsontext.Encoder, mask uint64, names []string) error {
	if len(names) < bits.Len64(mask) {
		return EncodingError(mask)
	}

	if err := enc.WriteToken(jsontext.BeginArray); err != nil {
		return err
	}

	for remaining := mask; remaining != 0; remaining &= remaining - 1 {
		pos := bits.TrailingZeros64(remaining)
		if err := enc.WriteToken(jsontext.String(names[pos])); err != nil {
			return err
		}
	}

	if err := enc.WriteToken(jsontext.EndArray); err != nil {
		return err
	}

	return nil
}

// UnmarshalJSONFrom parses a JSON encoding of a bitmask into a uint64. It accepts:
//   - a bool (only when trueValue != 0): "true" yields trueValue; "false" and
//     "[]" yield 0. Passing trueValue == 0 disables bool input.
//   - a string: parsed via [UnmarshalText] / [unmarshalTextString], so the
//     same comma- or whitespace-separated grammar applies as for plain text.
//   - an array of strings: each element is resolved via parseName and
//     accumulated, with a token resolving to 0 resetting the accumulator
//     (the same zero-reset semantic as [UnmarshalText]).
func UnmarshalJSONFrom(
	dec *jsontext.Decoder,
	parseName func(string) (uint64, bool),
	trueValue uint64,
) (uint64, error) {
	tok, err := dec.ReadToken()
	if err != nil {
		return 0, err
	}

	switch k := tok.Kind(); k {
	case jsontext.KindTrue:
		if trueValue == 0 {
			return 0, DecodingError(tok.String())
		}

		return trueValue, nil

	case jsontext.KindFalse:
		if trueValue == 0 {
			return 0, DecodingError(tok.String())
		}

		return 0, nil

	case jsontext.KindNull:
		return 0, nil

	case jsontext.KindString:
		return unmarshalTextString(tok.String(), parseName)

	case jsontext.KindBeginArray:
		return unmarshalJSONArrayFrom(dec, parseName)

	default:
		return 0, fmt.Errorf("cannot unmarshal %q", tok)
	}
}

// unmarshalJSONArrayFrom unmarshals an array of strings: each element is
// resolved via parseName and accumulated, with a token resolving to 0
// resetting the accumulator.
func unmarshalJSONArrayFrom(dec *jsontext.Decoder, parseName func(string) (uint64, bool)) (uint64, error) {
	var v uint64

	for {
		tok, err := dec.ReadToken()
		if err != nil {
			return 0, err
		}

		switch k := tok.Kind(); k {
		case jsontext.KindString:
			name := tok.String()

			mask, ok := parseName(name)
			if !ok {
				return 0, DecodingError(name)
			}

			v = accumulate(v, mask)

		case jsontext.KindEndArray:
			return v, nil

		default:
			return 0, fmt.Errorf("cannot unmarshal %q", tok)
		}
	}
}
