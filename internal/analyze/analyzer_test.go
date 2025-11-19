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

	. "fillmore-labs.com/scopeguard/internal/analyze"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()

	o := DefaultOptions()
	o.Generated = true

	analysistest.RunWithSuggestedFixes(t, testdata, o.Analyzer(), "./a")
}

func TestAnalyzerB(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()

	analysistest.Run(t, testdata, DefaultOptions().Analyzer(), "./b")
}
