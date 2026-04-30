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

func TestAppendText(t *testing.T) {
	t.Parallel()

	tests := [...]struct {
		name    string
		want    string
		wantErr bool
		value   uint64
	}{
		{name: "Zero value", value: 0, want: ""},
		{name: "Single value", value: 1, want: "one"},
		{name: "Two values", value: 1 | 4, want: "one, three"},
		{name: "Three values", value: 1 | 2 | 4, want: "one, two, three"},
		{name: "Unknown value", value: 8, wantErr: true},
		{name: "Mixed values", value: 1 | 8, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			txt, err := AppendText(nil, tt.value, testNames[:])
			if tt.wantErr {
				if !errors.As(err, new(EncodingError)) {
					t.Fatalf("AppendText() expected encoding error, got %v", err)
				}

				return
			}

			if err != nil {
				t.Fatalf("AppendText() error = %v", err)
			}

			if got := string(txt); got != tt.want {
				t.Errorf("AppendText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func BenchmarkAppendText(b *testing.B) {
	benchs := [...]struct {
		name  string
		buf   []byte
		value uint64
	}{
		{name: "nil/known", buf: nil, value: 1 | 2 | 4},
		{name: "pre/known", buf: make([]byte, 0, 16), value: 1 | 2 | 4},
		{name: "nil/unknown", buf: nil, value: 1 | 8},
	}
	for _, bb := range benchs {
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				_, _ = AppendText(bb.buf[:0], bb.value, testNames[:])
			}
		})
	}
}

func TestFormatText(t *testing.T) {
	t.Parallel()

	tests := [...]struct {
		name    string
		want    string
		wantErr bool
		value   uint64
	}{
		{name: "Zero value", value: 0, want: ""},
		{name: "Single value", value: 1, want: "one"},
		{name: "Last single value", value: 4, want: "three"},
		{name: "Two values", value: 1 | 4, want: "one, three"},
		{name: "Three values", value: 1 | 2 | 4, want: "one, two, three"},
		{name: "Unknown value", value: 8, wantErr: true},
		{name: "Mixed values", value: 1 | 8, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := FormatText(tt.value, testNames[:])
			if tt.wantErr {
				if !errors.As(err, new(EncodingError)) {
					t.Fatalf("FormatText() expected encoding error, got %v", err)
				}

				return
			}

			if err != nil {
				t.Fatalf("FormatText() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("FormatText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func BenchmarkFormatText(b *testing.B) {
	benchs := [...]struct {
		name  string
		value uint64
	}{
		{name: "single", value: 4},
		{name: "multi", value: 1 | 2 | 4},
		{name: "unknown", value: 1 | 8},
	}
	for _, bb := range benchs {
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				_, _ = FormatText(bb.value, testNames[:])
			}
		})
	}
}

var testNames = [...]string{
	"one",
	"two",
	"three",
}

func parseTestName(name string) (uint64, bool) {
	switch name {
	case "one":
		return 1, true
	case "two":
		return 2, true
	case "three":
		return 4, true
	case "zero":
		return 0, true
	case "all":
		return 7, true
	default:
		return 0, false
	}
}

func TestUnmarshalText(t *testing.T) {
	t.Parallel()

	tests := [...]struct {
		name string
		text string
		want uint64
		err  bool
	}{
		{name: "single", text: "one", want: 1},
		{name: "comma", text: "one, three", want: 1 | 4},
		{name: "whitespace", text: "one two\tthree", want: 1 | 2 | 4},
		{name: "alias", text: "all", want: 7},
		{name: "zero resets", text: "one, zero, two", want: 2},
		{name: "trailing zero", text: "all, zero", want: 0},
		{name: "empty", text: "", want: 0},
		{name: "unknown", text: "one, bogus", err: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := UnmarshalText([]byte(tt.text), parseTestName)
			if (err != nil) != tt.err {
				t.Fatalf("UnmarshalText() err = %v, want err = %t", err, tt.err)
			}

			if !tt.err && got != tt.want {
				t.Errorf("UnmarshalText() = %d, want %d", got, tt.want)
			}
		})
	}
}

func BenchmarkUnmarshalText(b *testing.B) {
	b.ReportAllocs()

	for text := []byte("one, three"); b.Loop(); {
		_, _ = UnmarshalText(text, parseTestName)
	}
}
