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

package gclplugin

import scopeguard "fillmore-labs.com/scopeguard/analyzer"

// Settings represents the configuration options for an instance of the [Plugin].
type Settings struct {
	// Scope enables scope checks.
	Scope *bool `json:"scope,omitzero"`
	// Shadow enables shadow checks.
	Shadow *bool `json:"shadow,omitzero"`
	// NestedAssign enables nested assignment checks.
	NestedAssign *bool `json:"nested-assign,omitzero"`
	// Conservative restricts moves to those without potential side effects.
	Conservative *bool `json:"conservative,omitzero"`
	// Combine enables combining declarations when moving to control flow initializers.
	Combine *bool `json:"combine,omitzero"`
	// Rename enables renaming of shadowed variables.
	Rename *bool `json:"rename,omitzero"`
	// MaxLines sets the maximum declaration size for moving to control flow initializers.
	MaxLines *int `json:"max-lines,omitzero"`
}

// Options converts [Settings] into a list of [scopeguard.Option] for the scopeguard analyzer.
// It processes settings and applies them only when explicitly set (non-nil).
func (s Settings) Options() []scopeguard.Option {
	var opts []scopeguard.Option

	opts = appendOption(opts, s.Scope, scopeguard.WithScope)
	opts = appendOption(opts, s.Shadow, scopeguard.WithShadow)
	opts = appendOption(opts, s.NestedAssign, scopeguard.WithNestedAssign)
	opts = appendOption(opts, s.Conservative, scopeguard.WithConservative)
	opts = appendOption(opts, s.Combine, scopeguard.WithCombine)
	opts = appendOption(opts, s.Rename, scopeguard.WithRename)
	opts = appendOption(opts, s.MaxLines, scopeguard.WithMaxLines)

	return opts
}

// appendOption appends a non-nil setting to a [scopeguard.Option] list.
func appendOption[T any](opts []scopeguard.Option, value *T, constructor func(T) scopeguard.Option) []scopeguard.Option {
	if value == nil {
		return opts
	}

	return append(opts, constructor(*value))
}
