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

package engine_test

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"testing"

	. "fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
)

var (
	keySink EditID
	idSink  string
)

func BenchmarkEditKey(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		editKey := EditKey{
			Filename: "/path/to/file.go",
			Message:  "Some diagnostic",
			Offset:   10,
		}

		keySink = editKey.EditID()
	}
}

func BenchmarkEditID(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		editID := EditID(0x01234567)

		idSink = editID.String()
	}
}

func TestEditID_String(t *testing.T) {
	t.Parallel()

	var id1, id2 uint32 = 0x01234567, 0x89abcdef

	s1 := EditID(id1).String()
	s2 := EditID(id2).String()

	if got, want := s1, fmt.Sprintf("%08x", id1); got != want {
		t.Errorf("want %q, got %q", want, got)
	}

	if got, want := s2, fmt.Sprintf("%08x", id2); got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestEditKey_EditID_AgreesWithHashFnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  EditKey
	}{
		{
			name: "simple ASCII",
			key: EditKey{
				Filename: "/path/to/file.go",
				Message:  "Some diagnostic",
				Offset:   10,
			},
		},
		{
			name: "empty values",
			key: EditKey{
				Filename: "",
				Message:  "",
				Offset:   0,
			},
		},
		{
			name: "negative offset",
			key: EditKey{
				Filename: "test.go",
				Message:  "error",
				Offset:   -1,
			},
		},
		{
			name: "non-ASCII characters",
			key: EditKey{
				Filename: "/path/to/üñïçødê_file.go",
				Message:  "Diagnostic with non-ASCII: ★☆",
				Offset:   123456,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Compute standard FNV-1a hash
			expected := fnv32a(tc.key)
			if expected == 0 {
				expected = 1
			}

			if got, want := tc.key.EditID(), EditID(expected); got != want {
				t.Errorf("EditID() = %v, want %v", got, want)
			}
		})
	}
}

func fnv32a(key EditKey) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key.Filename))
	_, _ = h.Write([]byte{':'})
	_, _ = h.Write([]byte(strconv.Itoa(key.Offset)))
	_, _ = h.Write([]byte{':'})
	_, _ = h.Write([]byte(key.Message))

	return h.Sum32()
}
