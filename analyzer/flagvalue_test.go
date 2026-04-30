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

package analyzer_test

import (
	"flag"
	"strings"
	"testing"

	. "fillmore-labs.com/scopeguard/analyzer"
)

type testFlags uint8

const (
	flagTest testFlags = 1 << iota
	noFlags  testFlags = 0
)

func (t *testFlags) Set(flag testFlags, value bool) {
	if value {
		*t |= flag
	} else {
		*t &^= flag
	}
}

func (t testFlags) Enabled(flag testFlags) bool { return t&flag != 0 }

func TestFlagValue(t *testing.T) {
	t.Parallel()

	tests := [...]struct {
		name    string
		args    []string
		initial testFlags
		want    bool
	}{
		{
			name:    "Enable",
			initial: noFlags,
			args:    []string{"-flag"},
			want:    true,
		},
		{
			name:    "Disable",
			initial: flagTest,
			args:    []string{"-flag=false"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			flags := tt.initial

			fs := flag.NewFlagSet("test", flag.ContinueOnError)

			fv := NewFlagValue(&flags, flagTest)
			fs.Var(fv, "flag", "enable flag")

			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if fv.Get() != tt.want {
				t.Errorf("Flag get = %v, want %v", fv.Get(), tt.want)
			}

			if flags&flagTest != 0 != tt.want {
				t.Errorf("Flag enabled = %t, want %t", flags&flagTest != 0, tt.want)
			}
		})
	}
}

func TestUsage(t *testing.T) {
	t.Parallel()

	flags := flagTest

	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	fv := NewFlagValue(&flags, flagTest)
	fs.Var(fv, "flag", "enable flag")

	const expectedUsage = `
  -flag
    	enable flag (default true)
`

	var out strings.Builder
	fs.SetOutput(&out)
	fs.Usage()

	if got, want := out.String(), expectedUsage; !strings.HasSuffix(got, want) {
		t.Errorf("Usage() = %q, want suffix %q", got, want)
	}
}
