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

package run

import (
	"go/token"
	"reflect"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"fillmore-labs.com/scopeguard/internal/typeutil"
)

// Public API constants for the scopeguard analyzer.
const (
	name = "scopeguard"
	doc  = `scopeguard detects variables that can be moved to tighter scopes`
	url  = "https://pkg.go.dev/fillmore-labs.com/scopeguard"
)

// Analyzer returns a configured scopeguard analyzer.
func (o *Options) Analyzer() *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name:       name,
		Doc:        doc,
		URL:        url,
		Run:        o.run,
		Requires:   []*analysis.Analyzer{inspect.Analyzer},
		ResultType: reflect.TypeFor[Result](),
	}

	o.registerFlags(&a.Flags)

	return a
}

// Result is the internal analyzer result.
type Result struct {
	// Processed contains the matched functions when a filter is set.
	Processed []Function
}

// Function is the range of a function or method with fixes in the source code.
type Function struct {
	Name typeutil.LocalFuncName
	Pos  token.Pos
	End  token.Pos
}
