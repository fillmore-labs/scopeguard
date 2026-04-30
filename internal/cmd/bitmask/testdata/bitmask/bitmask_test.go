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
	"bytes"
	"encoding/json"
	"testing"

	. "fillmore-labs.com/scopeguard/internal/cmd/bitmask/testdata/bitmask"
)

func TestBitmask_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		val  Bitmask
		want string
	}{
		{None, "zero"},
		{One, "one"},
		{Two, "two"},
		{Three, "three"},
		{All, "all"},
		{One | Two, "one, two"},
		{One | Three, "one, three"},
		{Two | Three, "two, three"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			if got := tt.val.String(); got != tt.want {
				t.Errorf("Bitmask.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBitmask_Text(t *testing.T) {
	t.Parallel()

	tests := []struct {
		val  Bitmask
		text string
	}{
		{None, "zero"},
		{One, "one"},
		{Two, "two"},
		{Three, "three"},
		{All, "all"},
		{One | Two, "one, two"},
		{One | Three, "one, three"},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			t.Parallel()

			// Marshal
			buf, err := tt.val.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText() unexpected error: %v", err)
			}

			if got := string(buf); got != tt.text {
				t.Errorf("MarshalText() = %q, want %q", got, tt.text)
			}

			// Unmarshal
			var got Bitmask
			if err := got.UnmarshalText(buf); err != nil {
				t.Fatalf("UnmarshalText() unexpected error: %v", err)
			}

			if got != tt.val {
				t.Errorf("UnmarshalText() = %v, want %v", got, tt.val)
			}
		})
	}
}

func TestBitmask_TextError(t *testing.T) {
	t.Parallel()

	// Test unmarshal error
	var dummy Bitmask
	if err := dummy.UnmarshalText([]byte("invalid")); err == nil {
		t.Error("UnmarshalText() expected error for invalid input, got nil")
	}
}

func TestBitmask_JSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		jsonStr string
		want    Bitmask
	}{
		{"bool true", "true", All},
		{"bool false", "false", None},
		{"string single", `"one"`, One},
		{"string comma", `"one, two"`, One | Two},
		{"array empty", `[]`, None},
		{"array", `["one", "three"]`, One | Three},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			buf := bytes.NewBufferString(tt.jsonStr)
			dec := json.NewDecoder(buf)

			var got *Bitmask
			if err := dec.Decode(&got); err != nil {
				t.Fatalf("json.Unmarshal(%s) error: %v", tt.jsonStr, err)
			}

			if got == nil || *got != tt.want {
				t.Errorf("json.Unmarshal(%s) = %v, want %v", tt.jsonStr, got, tt.want)
			}
		})
	}
}

// TestFlag_String exercises the helpers generated for Flag, a type declared in
// this external test package. Its code is generated into flag_bitmask_test.go.
func TestFlag_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		val  Flag
		want string
	}{
		{FlagRead, "read"},
		{FlagWrite, "write"},
		{FlagRead | FlagWrite, "read, write"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			if got := tt.val.String(); got != tt.want {
				t.Errorf("Flag.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBitmask_JSONErrors(t *testing.T) {
	t.Parallel()

	// Test unmarshal errors
	tests := []struct {
		name    string
		jsonStr string
	}{
		{"invalid json", "{invalid}"},
		{"invalid type", "123"},
		{"unknown bitmask string", `"invalid"`},
		{"unknown array element", `["one", "invalid"]`},
		{"non-string array element", `["one", true]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			buf := bytes.NewBufferString(tt.jsonStr)
			dec := json.NewDecoder(buf)

			var got *Bitmask
			if err := dec.Decode(&got); err == nil {
				t.Errorf("json.Unmarshal(%s) expected error, got nil", tt.jsonStr)
			}
		})
	}
}
