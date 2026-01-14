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
	"fillmore-labs.com/scopeguard/internal/config"
)

func TestFlagValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		initial config.AnalyzerFlags
		args    []string
		want    bool
	}{
		{
			name:    "Enable",
			initial: config.ShadowAnalyzer,
			args:    []string{"-scope"},
			want:    true,
		},
		{
			name:    "Disable",
			initial: config.ScopeAnalyzer,
			args:    []string{"-scope=false"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var flags config.Analyzers
			flags.Set(tt.initial, true)

			fs := flag.NewFlagSet("test", flag.ContinueOnError)

			const value = config.ScopeAnalyzer
			fv := NewAnalyzerValue(&flags, value)
			fs.Var(fv, "scope", "enable scope analyzer")

			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if fv.Get() != tt.want {
				t.Errorf("Flag get = %v, want %v", fv.Get(), tt.want)
			}

			if flags.Enabled(value) != tt.want {
				t.Errorf("ScopeAnalyzer enabled = %v, want %v", flags.Enabled(config.ScopeAnalyzer), tt.want)
			}
		})
	}
}

func TestUsage(t *testing.T) {
	t.Parallel()

	var flags config.Analyzers
	flags.Set(config.ScopeAnalyzer, true)

	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	fv := NewAnalyzerValue(&flags, config.ScopeAnalyzer)
	fs.Var(fv, "scope", "enable scope analyzer")

	const expectedUsage = `
  -scope
    	enable scope analyzer (default true)
`

	var out strings.Builder
	fs.SetOutput(&out)
	fs.Usage()

	if got, want := out.String(), expectedUsage; !strings.HasSuffix(got, want) {
		t.Errorf("Usage() = %q, want suffix %q", got, want)
	}
}
