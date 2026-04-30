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
	"fillmore-labs.com/scopeguard/internal/config"
	"fillmore-labs.com/scopeguard/internal/typeutil"
)

// Options represent configuration runOptions for the scopeguard analyzer.
type Options struct {
	Renames typeutil.RenameMap

	// Functions optionally restricts analysis to the named functions and methods.
	Functions []typeutil.LocalFuncName

	// MaxLines specifies the maximum number of lines a declaration can span to be considered for moving
	// into control flow initializers.
	MaxLines int

	// Analyzers represent the Analyzers to be enabled.
	Analyzers config.Analyzers

	// Behaviors holds layout and behavioral options.
	Behaviors config.Behaviors

	// Filters filters diagnostics by safety.
	Filters config.Filters
}

// DefaultOptions initializes and returns a new Options instance with default values.
func DefaultOptions() *Options {
	return &Options{
		MaxLines:  -1,
		Analyzers: config.DefaultAnalyzers(),
		Behaviors: config.DefaultBehavior(),
		Filters:   config.NewFilters(config.FilterAll, config.FilterSafe),
	}
}
