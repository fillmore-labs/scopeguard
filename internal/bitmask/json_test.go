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

package bitmask_test

import (
	"errors"
	"testing"

	. "fillmore-labs.com/scopeguard/internal/bitmask"
)

func TestMarshalJSON(t *testing.T) {
	t.Parallel()

	tests := [...]struct {
		name  string
		want  string
		value uint64
	}{
		{name: "Zero value", value: 0, want: "[]"},
		{name: "Single value", value: 1, want: "[\"one\"]"},
		{name: "Two values", value: 1 | 4, want: "[\"one\",\"three\"]"},
		{name: "Three values", value: 1 | 2 | 4, want: "[\"one\",\"two\",\"three\"]"},
		{name: "Unknown value", value: 8},
		{name: "Mixed values", value: 1 | 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			switch txt, err := MarshalJSON(tt.value, testNames[:]); tt.want {
			case "":
				if !errors.As(err, new(EncodingError)) {
					t.Fatalf("MarshalJSON() expected encoding error, got %v", err)
				}

			default:
				if err != nil {
					t.Fatalf("MarshalJSON() error = %v", err)
				}

				if got := string(txt); got != tt.want {
					t.Errorf("MarshalJSON() = %q, want %q", got, tt.want)
				}
			}
		})
	}
}

func BenchmarkMarshalJSON(b *testing.B) {
	benchs := [...]struct {
		name  string
		value uint64
	}{
		{name: "zero", value: 0},
		{name: "known", value: 1 | 2 | 4},
		{name: "unknown", value: 1 | 8},
	}
	for _, bb := range benchs {
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				_, _ = MarshalJSON(bb.value, testNames[:])
			}
		})
	}
}

func TestUnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := [...]struct {
		name string
		data string
		want uint64
		err  bool
	}{
		{name: "bool true", data: "true", want: 7},
		{name: "bool false", data: "false", want: 0},
		{name: "bracket empty", data: "[]", want: 0},
		{name: "string", data: `"one, three"`, want: 1 | 4},
		{name: "array", data: `["one", "three"]`, want: 1 | 4},
		{name: "array with zero resets", data: `["one", "zero", "two"]`, want: 2},
		{name: "string escape", data: `"one"`, want: 1},
		{name: "string tab escape", data: `"one\ttwo"`, want: 1 | 2},
		{name: "string newline escape", data: `"one\nthree"`, want: 1 | 4},
		{name: "string unicode escape", data: `"\u006f\u006e\u0065"`, want: 1},
		{name: "array unicode escape", data: `["\u006fne", "two"]`, want: 1 | 2},
		{name: "array element is not split", data: `["one\ttwo"]`, err: true},
		{name: "string invalid escape", data: `"on\\e"`, err: true},
		{name: "array invalid escape", data: `["on\\e"]`, err: true},
		{name: "unknown name", data: `"bogus"`, err: true},
		{name: "unknown in array", data: `["one", "bogus"]`, err: true},
		{name: "non-string array elem", data: `["one", 5]`, err: true},
		{name: "invalid json", data: `{`, err: true},
		{name: "wrong type", data: `42`, err: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := UnmarshalJSON([]byte(tt.data), parseTestName, 7)
			if (err != nil) != tt.err {
				t.Fatalf("UnmarshalJSON(%q) err = %v, want err=%v", tt.data, err, tt.err)
			}

			if !tt.err && got != tt.want {
				t.Errorf("UnmarshalJSON(%q) = %d, want %d", tt.data, got, tt.want)
			}
		})
	}
}

func TestUnmarshalJSON_NoHasTrue(t *testing.T) {
	t.Parallel()

	// Without hasTrue, "true" and "false" must not be special-cased.
	if _, err := UnmarshalJSON([]byte("true"), parseTestName, 0); err == nil {
		t.Error("UnmarshalJSON(true) without hasTrue: expected error, got nil")
	}

	// "[]" still works via the JSON fallback (empty array → 0).
	got, err := UnmarshalJSON([]byte("[]"), parseTestName, 0)
	if err != nil {
		t.Fatalf("UnmarshalJSON([]) without hasTrue: unexpected error: %v", err)
	}

	if got != 0 {
		t.Errorf("UnmarshalJSON([]) = %d, want 0", got)
	}
}

func BenchmarkUnmarshalJSON(b *testing.B) {
	benchs := [...]struct {
		name string
		data string
	}{
		{name: "bool", data: `true`},
		{name: "string", data: `"one, three"`},
		{name: "array", data: `["one", "three"]`},
	}
	for _, bb := range benchs {
		data := []byte(bb.data)
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				_, _ = UnmarshalJSON(data, parseTestName, 7)
			}
		})
	}
}
