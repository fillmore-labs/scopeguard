// Copyright 2025 Oliver Eikemeier. All Rights Reserved.
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

package analyze_test

import (
	"testing"

	"fillmore-labs.com/scopeguard/analyzer/level"
	. "fillmore-labs.com/scopeguard/internal/analyze"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()

	tests := []struct {
		name    string
		dir     string
		options func(*Options)
		fix     bool
	}{
		{
			name: "Default",
			dir:  "./a",
			options: func(o *Options) {
				o.Generated = true
				o.MaxLines = 5
			},
			fix: true,
		},
		{
			name: "NoFix",
			dir:  "./b",
		},
		{
			name: "Conservative",
			dir:  "./c",
			options: func(o *Options) {
				o.ScopeLevel = level.ScopeConservative
			},
			fix: true,
		},
		{
			name: "Combine",
			dir:  "./d",
			options: func(o *Options) {
				o.Combine = true
			},
			fix: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			o := DefaultOptions()
			if tt.options != nil {
				tt.options(o)
			}

			if tt.fix {
				analysistest.RunWithSuggestedFixes(t, testdata, o.Analyzer(), tt.dir)
			} else {
				analysistest.Run(t, testdata, o.Analyzer(), tt.dir)
			}
		})
	}
}
