// Copyright 2025-2026 Oliver Eikemeier. All Rights Reserved.
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
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	. "fillmore-labs.com/scopeguard/analyzer"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()

	tests := []struct {
		name    string
		dir     string
		options Option
		fix     bool
	}{
		{
			name:    "Default",
			dir:     "./a",
			options: Options{WithGenerated(true), WithMaxLines(5)},
			fix:     true,
		},
		{
			name: "NoFix",
			dir:  "./nofix",
		},
		{
			name:    "Conservative",
			dir:     "./conservative",
			options: Options{WithConservative(true), WithCombine(false)},
			fix:     true,
		},
		{
			name:    "Combine",
			dir:     "./combine",
			options: WithCombine(true),
			fix:     true,
		},
		{
			name:    "Rename",
			dir:     "./rename",
			options: Options{WithScope(false), WithNestedAssign(false), WithRename(true)},
			fix:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if a := New(tt.options); tt.fix {
				analysistest.RunWithSuggestedFixes(t, testdata, a, tt.dir)
			} else {
				analysistest.Run(t, testdata, a, tt.dir)
			}
		})
	}
}
