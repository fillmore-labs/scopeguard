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

package analyze

// Options represent configuration options for the scopeguard analyzer.
type Options struct {
	// Analyzers represents the analyzers to be enabled.
	Analyzers BitMask[Analyzer]

	// Behavior holds layout and behavioral options.
	Behavior BitMask[Config]

	// MaxLines specifies the maximum number of lines a declaration can span to be considered for moving
	// into control flow initializers.
	MaxLines int
}

// DefaultOptions initializes and returns a new Options instance with default values.
func DefaultOptions() *Options {
	o := &Options{
		Analyzers: BitMask[Analyzer]{value: ScopeAnalyzer | ShadowAnalyzer | NestedAssignAnalyzer},
		Behavior:  BitMask[Config]{value: CombineDecls},
		MaxLines:  -1,
	}

	return o
}

// Analyzer represents specific analyzers.
type Analyzer uint8

const (
	// ScopeAnalyzer enables scope-based analysis for identifying variable declarations and usage.
	ScopeAnalyzer Analyzer = 1 << iota

	// ShadowAnalyzer enables analysis to detect shadowed variable declarations.
	ShadowAnalyzer

	// NestedAssignAnalyzer enables the analysis of nested assignments.
	NestedAssignAnalyzer
)

// Config represents configuration options for the analyzers.
type Config uint8

const (
	// IncludeGenerated specifies whether to include analysis of generated files.
	IncludeGenerated Config = 1 << iota

	// CombineDecls determines whether to combine declarations when moving to init statements.
	CombineDecls

	// Conservative indicates that moves should be conservative.
	Conservative

	// RenameVars indicates that shadowed variables should be renamed.
	RenameVars
)

// BitMask is a generic type that represents a bitmask for managing binary flags.
type BitMask[T ~uint8] struct { // constraints.Integer would be fine, but it lives in golang.org/x/exp
	value T
}

// Set adjusts the bitmask by enabling or disabling the specified option.
func (o *BitMask[T]) Set(flag T, value bool) {
	if value {
		o.Enable(flag)
	} else {
		o.Disable(flag)
	}
}

// Enable sets the given flag in the current bitmask, enabling the specified option.
func (o *BitMask[T]) Enable(flag T) {
	o.value |= flag
}

// Disable removes the specified flag from the current bitmask, disabling the associated option.
func (o *BitMask[T]) Disable(flag T) {
	o.value &^= flag
}

// Enabled checks if the specified option is enabled in the current bitmask.
func (o BitMask[T]) Enabled(flag T) bool {
	return o.value&flag != 0
}
