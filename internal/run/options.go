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

import "fillmore-labs.com/scopeguard/internal/config"

// Options represent configuration runOptions for the scopeguard analyzer.
type Options struct {
	// Analyzers represent the Analyzers to be enabled.
	Analyzers config.Analyzers

	// Behavior holds layout and behavioral options.
	Behavior config.Behavior

	// MaxLines specifies the maximum number of lines a declaration can span to be considered for moving
	// into control flow initializers.
	MaxLines int
}

// DefaultOptions initializes and returns a new Options instance with default values.
func DefaultOptions() *Options {
	return &Options{
		Analyzers: config.DefaultAnalyzers(),
		Behavior:  config.DefaultBehavior(),
		MaxLines:  -1,
	}
}
