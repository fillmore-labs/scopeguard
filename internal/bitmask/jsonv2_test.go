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

package bitmask_test

import (
	"bytes"
	"encoding/json/jsontext"
	"errors"
	"io"
	"strings"
	"testing"

	. "fillmore-labs.com/scopeguard/internal/bitmask"
)

func TestMarshalJSONTo(t *testing.T) {
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

			var buf bytes.Buffer
			enc := jsontext.NewEncoder(&buf)

			switch err := MarshalJSONTo(enc, tt.value, testNames[:]); tt.want {
			case "":
				if !errors.As(err, new(EncodingError)) {
					t.Fatalf("MarshalJSONTo() expected encoding error, got %v", err)
				}

			default:
				if err != nil {
					t.Fatalf("MarshalJSONTo() error = %v", err)
				}

				if got := buf.String(); got != tt.want && got != tt.want+"\n" {
					t.Errorf("MarshalJSONTo() = %q, want %q", got, tt.want)
				}
			}
		})
	}
}

func TestUnmarshalJSONFrom(t *testing.T) {
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
		{name: "unknown name", data: `"bogus"`, err: true},
		{name: "unknown in array", data: `["one", "bogus"]`, err: true},
		{name: "non-string array elem", data: `["one", 5]`, err: true},
		{name: "invalid json", data: `{`, err: true},
		{name: "wrong type", data: `42`, err: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dec := jsontext.NewDecoder(strings.NewReader(tt.data))

			got, err := UnmarshalJSONFrom(dec, parseTestName, 7)
			if (err != nil) != tt.err {
				t.Fatalf("UnmarshalJSONFrom(%q) err = %v, want err=%v", tt.data, err, tt.err)
			}

			if !tt.err && got != tt.want {
				t.Errorf("UnmarshalJSONFrom(%q) = %d, want %d", tt.data, got, tt.want)
			}
		})
	}
}

func BenchmarkUnmarshalJSONFrom(b *testing.B) {
	benchs := [...]struct {
		name string
		data string
	}{
		{name: "bool", data: `true`},
		{name: "string", data: `"one, three"`},
		{name: "array", data: `["one", "three"]`},
	}
	for _, bb := range benchs {
		r := strings.NewReader(bb.data)
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				_, _ = r.Seek(0, io.SeekStart)
				dec := jsontext.NewDecoder(r)
				_, _ = UnmarshalJSONFrom(dec, parseTestName, 7)
			}
		})
	}
}
