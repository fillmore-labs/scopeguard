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

package analyzer

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"fillmore-labs.com/scopeguard/internal/config"
)

// runOptions represent configuration runOptions for the scopeguard analyzer.
type runOptions struct {
	// analyzers represents the analyzers to be enabled.
	analyzers config.BitMask[config.AnalyzerFlags]

	// behavior holds layout and behavioral options.
	behavior config.BitMask[config.Config]

	// maxLines specifies the maximum number of lines a declaration can span to be considered for moving
	// into control flow initializers.
	maxLines int
}

// makeRunOptions returns a [options] struct with overriding [Options] applied.
func makeRunOptions(opts Options) *runOptions {
	r := defaultRunOptions()
	opts.apply(r)

	return r
}

// defaultRunOptions initializes and returns a new Options instance with default values.
func defaultRunOptions() *runOptions {
	return &runOptions{
		analyzers: config.NewBitMask(config.ScopeAnalyzer | config.ShadowAnalyzer | config.NestedAssignAnalyzer),
		behavior:  config.NewBitMask(config.CombineDeclarations),
		maxLines:  -1,
	}
}

// analyzer returns a scopeguard *[analysis.analyzer] instance.
func (r *runOptions) analyzer() *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name:     name,
		Doc:      doc,
		URL:      url,
		Run:      r.run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}

	return a
}
