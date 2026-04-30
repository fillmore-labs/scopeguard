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
	"strings"
	"testing"

	. "fillmore-labs.com/scopeguard/internal/cmd/bitmask/testdata/bitmask"
)

func TestBitmask_JSONv2(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  Bitmask
		want string
	}{
		{"None", None, "[]"},
		{"One", One, "[\"one\"]"},
		{"Two", Two, "[\"two\"]"},
		{"Three", Three, "[\"three\"]"},
		{"Combo", One | Three, "[\"one\",\"three\"]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			enc := jsontext.NewEncoder(&buf)
			if err := tt.val.MarshalJSONTo(enc); err != nil {
				t.Fatalf("MarshalJSONTo() unexpected error: %v", err)
			}

			got := strings.TrimSpace(buf.String())
			if got != tt.want {
				t.Errorf("MarshalJSONTo() = %q, want %q", got, tt.want)
			}

			// Test roundtrip
			dec := jsontext.NewDecoder(strings.NewReader(got))

			var val Bitmask
			if err := (&val).UnmarshalJSONFrom(dec); err != nil {
				t.Fatalf("UnmarshalJSONFrom() unexpected error: %v", err)
			}

			if val != tt.val {
				t.Errorf("UnmarshalJSONFrom() = %v, want %v", val, tt.val)
			}
		})
	}
}
