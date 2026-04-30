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
	"flag"
	"testing"

	. "fillmore-labs.com/scopeguard/internal/cmd/bitmask/testdata/bitmask"
)

// bindMode wires m.BoolFlag(mask) into a fresh FlagSet under name and returns it.
func bindMode(m *Mode, mask Mode, name string) *flag.FlagSet {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.Var(m.BoolFlag(mask), name, "usage")

	return fs
}

func TestMode_BoolFlag_Default(t *testing.T) {
	t.Parallel()

	var m Mode

	fs := bindMode(&m, ModeWrite, "write")
	if got := fs.Lookup("write").Value.String(); got != "false" {
		t.Errorf("unset flag String() = %q, want %q", got, "false")
	}
}

func TestMode_BoolFlag_Set(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want Mode
	}{
		{"bare", []string{"-write"}, ModeWrite},
		{"explicit true", []string{"-write=true"}, ModeWrite},
		{"explicit false", []string{"-write=false"}, 0},
		{"absent", nil, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var m Mode

			fs := bindMode(&m, ModeWrite, "write")
			if err := fs.Parse(tc.args); err != nil {
				t.Fatalf("Parse(%q): %v", tc.args, err)
			}

			if m != tc.want {
				t.Errorf("Parse(%q): Mode = %v, want %v", tc.args, m, tc.want)
			}

			if got, want := m.Has(ModeWrite), tc.want == ModeWrite; got != want {
				t.Errorf("Parse(%q): Has(ModeWrite) = %t, want %t", tc.args, got, want)
			}
		})
	}
}

// TestMode_BoolFlag_IsBoolFlag confirms `-write` toggles only its own bit and
// leaves an already-set sibling untouched (a non-bool flag would consume the
// next argument instead).
func TestMode_BoolFlag_IsBoolFlag(t *testing.T) {
	t.Parallel()

	m := ModeRead

	fs := bindMode(&m, ModeWrite, "write")
	if err := fs.Parse([]string{"-write", "arg"}); err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if !m.Has(ModeRead) || !m.Has(ModeWrite) {
		t.Errorf("Mode = %v, want read|write", m)
	}

	if got := fs.Args(); len(got) != 1 || got[0] != "arg" {
		t.Errorf("residual args = %v, want [arg]", got)
	}
}
