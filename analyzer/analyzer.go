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

package analyzer

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"fillmore-labs.com/scopeguard/internal/run"
)

// Public API constants for the scopeguard analyzer.
const (
	name = "scopeguard"
	doc  = `scopeguard detects variables that can be moved to tighter scopes`
	url  = "https://pkg.go.dev/fillmore-labs.com/scopeguard"
)

// New creates a new instance of the scopeguard analyzer.
// It allows for programmatic configuration using [Option], which is useful
// for integrating the analyzer into other tools. For command-line use, the
// pre-configured [Analyzer] variable is typically sufficient.
func New(opts ...Option) *analysis.Analyzer {
	r := run.DefaultOptions()
	Options(opts).apply(r)

	a := &analysis.Analyzer{
		Name:     name,
		Doc:      doc,
		URL:      url,
		Run:      r.Run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}

	registerFlags(&a.Flags, r)

	return a
}

// Analyzer is a pre-configured *[analysis.Analyzer] for detecting variables that can be moved to tighter scopes.
var Analyzer = New()
