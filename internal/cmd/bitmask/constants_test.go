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

package main

import "testing"

// constNameFor builds a [constinfo] whose serialized name is its identifier.
func constNameFor(name string, value uint64) constInfo {
	return constInfo{Const: name, Name: name, value: value}
}

// threeBits is a contiguous bit table (positions 0..2) shared by the
// buildAliases cases; buildAliases only reads its length.
var threeBits = map[int]constInfo{
	0: constNameFor("A", 1),
	1: constNameFor("B", 2),
	2: constNameFor("C", 4),
}

func TestBuildAliases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		combos  []constInfo
		wantErr bool
	}{
		{name: "combination in range", combos: []constInfo{constNameFor("All", 0b111)}},
		{name: "zero is in range", combos: []constInfo{constNameFor("None", 0)}},
		{name: "bit beyond defined bits", combos: []constInfo{constNameFor("Weird", 0b1011)}, wantErr: true},
		{
			name:    "duplicate value",
			combos:  []constInfo{constNameFor("X", 0b011), constNameFor("Y", 0b011)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &typeConstants{single: threeBits, combos: tt.combos}

			if _, err := c.buildAliases(); (err != nil) != tt.wantErr {
				t.Fatalf("buildAliases() err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestBuildAliasesValue verifies the parsed value is threaded onto each alias.
func TestBuildAliasesValue(t *testing.T) {
	t.Parallel()

	c := &typeConstants{single: threeBits, combos: []constInfo{constNameFor("All", 0b111)}}

	aliases, err := c.buildAliases()
	if err != nil {
		t.Fatalf("buildAliases() unexpected error: %v", err)
	}

	if len(aliases) != 1 {
		t.Fatalf("buildAliases() returned %d aliases, want 1", len(aliases))
	}

	if got := aliases[0]; got.Const != "All" || got.value != 0b111 {
		t.Errorf("buildAliases() = %+v, want Const=All Value=7", got)
	}
}
