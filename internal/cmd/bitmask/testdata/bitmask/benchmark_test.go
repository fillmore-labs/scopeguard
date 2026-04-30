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

package bitmask_test

import (
	"encoding/json"
	"testing"

	. "fillmore-labs.com/scopeguard/internal/cmd/bitmask/testdata/bitmask"
)

func BenchmarkBitmask_MarshalText(b *testing.B) {
	b.ReportAllocs()

	for s := One | Three; b.Loop(); {
		_, _ = s.MarshalText()
	}
}

func BenchmarkBitmask_AppendText(b *testing.B) {
	b.ReportAllocs()

	buf := make([]byte, 0, 64) // We overwrite this buffer repeatedly. Fine for a benchmark.

	for s := One | Three; b.Loop(); {
		_, _ = s.AppendText(buf) // ignore error
	}
}

func BenchmarkBitmask_String(b *testing.B) {
	benchs := [...]struct {
		name string
		val  Bitmask
	}{
		{"None", None},
		{"One", One},
		{"All", All},
		{"Combo", One | Three},
	}

	for _, bb := range benchs {
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				_ = bb.val.String()
			}
		})
	}
}

func BenchmarkBitmask_UnmarshalText(b *testing.B) {
	b.ReportAllocs()

	for text := []byte("one, three"); b.Loop(); {
		var s Bitmask
		_ = s.UnmarshalText(text) // ignore error
	}
}

func BenchmarkBitmask_MarshalJSON(b *testing.B) {
	benchs := [...]struct {
		name string
		val  Bitmask
	}{
		{"None", None},
		{"One", One},
		{"All", All},
		{"Combo", One | Three},
	}

	for _, bb := range benchs {
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				_, _ = json.Marshal(bb.val) // ignore error
			}
		})
	}
}

func BenchmarkBitmask_UnmarshalJSON(b *testing.B) {
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
				var mask Bitmask
				_ = json.Unmarshal(data, &mask) // ignore error
			}
		})
	}
}
