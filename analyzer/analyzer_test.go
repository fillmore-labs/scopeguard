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
	"fillmore-labs.com/scopeguard/internal/run"
	"fillmore-labs.com/scopeguard/internal/set"
	"fillmore-labs.com/scopeguard/internal/typeutil"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()

	tests := [...]struct {
		name    string
		options Option
		dir     string
		fix     bool
	}{
		{
			name:    "Default",
			dir:     "./a",
			options: Join(WithUnsafe(true), WithGenerated(true), WithMaxLines(5), WithRename(false)),
			fix:     true,
		},
		{
			name: "NoFix",
			dir:  "./nofix",
		},
		{
			name:    "Safe",
			dir:     "./safe",
			options: Join(WithUnsafeDiagnostics(false), WithCombine(false)),
			fix:     true,
		},
		{
			name:    "Combine",
			dir:     "./combine",
			options: WithCombine(true),
			fix:     true,
		},
		{
			name:    "Shadow",
			dir:     "./shadow",
			options: Join(WithScope(false), WithNestedAssign(false), WithRename(false)),
			fix:     true,
		},
		{
			name:    "Rename",
			dir:     "./rename",
			options: Join(WithScope(false), WithNestedAssign(false), WithRename(true), WithRenames(typeutil.RenameMap{{Name: "renameSecondNamed"}: {"r": {"s"}}})),
			fix:     true,
		},
		{
			name:    "RenameOnly",
			dir:     "./renameonly",
			options: Join(WithScope(false), WithNestedAssign(false), WithRename(false), WithRenames(typeutil.RenameMap{{Name: "renameSecondNamed"}: {"x": {"y"}}})),
			fix:     true,
		},
		{
			name:    "Filter",
			dir:     "./filter",
			options: WithFunctions(typeutil.LocalFuncName{Receiver: "a", Name: "A"}),
			fix:     true,
		},
		{
			name:    "Unsafe",
			dir:     "./unsafe",
			options: WithUnsafe(true),
			fix:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a, err := New(tt.options)
			if err != nil {
				t.Fatal(err)
			}

			if tt.fix {
				analysistest.RunWithSuggestedFixes(t, testdata, a, tt.dir)
			} else {
				analysistest.Run(t, testdata, a, tt.dir)
			}
		})
	}
}

func TestUnmatchedFilter(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()

	matchedFunc := typeutil.LocalFuncName{Receiver: "a", Name: "A"}
	unmatchedFunc := typeutil.LocalFuncName{Name: "Func"}

	functions := []typeutil.LocalFuncName{matchedFunc, unmatchedFunc}

	a, err := New(WithFunctions(functions...))
	if err != nil {
		t.Fatal(err)
	}

	results := analysistest.Run(t, testdata, a, "./filter")

	processed := set.New[typeutil.LocalFuncName]()

	for _, r := range results {
		for _, p := range r.Result.(run.Result).Processed {
			processed.Add(p.Name)
		}
	}

	if !processed.Contains(matchedFunc) {
		t.Errorf("Expected %s to be processed", matchedFunc)
	}

	if processed.Contains(unmatchedFunc) {
		t.Errorf("Expected %s to not be processed", unmatchedFunc)
	}
}
