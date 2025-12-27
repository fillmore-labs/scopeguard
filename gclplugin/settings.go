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

import (
	scopeguard "fillmore-labs.com/scopeguard/analyzer"
)

// Settings represent the configuration options for an instance of the [Plugin].
type Settings struct {
	Scope        *bool `json:"scope,omitzero"`
	Shadow       *bool `json:"shadow,omitzero"`
	NestedAssign *bool `json:"nested-assign,omitzero"`
	Conservative *bool `json:"conservative,omitzero"`
	Combine      *bool `json:"combine,omitzero"`
	Rename       *bool `json:"rename,omitzero"`
	MaxLines     *int  `json:"max-lines,omitzero"`
}

// Options converts [Settings] into [scopeguard.Options] for the scopeguard analyzer.
// It processes settings and applies them only when explicitly set (non-nil).
func (s Settings) Options() scopeguard.Options {
	var opts scopeguard.Options

	for _, options := range []func(scopeguard.Options) scopeguard.Options{
		mappings[bool]{
			{scopeguard.WithScope, s.Scope},
			{scopeguard.WithShadow, s.Shadow},
			{scopeguard.WithNestedAssign, s.NestedAssign},
			{scopeguard.WithConservative, s.Conservative},
			{scopeguard.WithCombine, s.Combine},
			{scopeguard.WithRename, s.Rename},
		}.options,
		mappings[int]{
			{scopeguard.WithMaxLines, s.MaxLines},
		}.options,
	} {
		opts = options(opts)
	}

	return opts
}

type mappings[T any] []struct {
	fun   func(T) scopeguard.Option
	value *T
}

func (m mappings[T]) options(opts scopeguard.Options) scopeguard.Options {
	for _, opt := range m {
		if opt.value == nil {
			continue
		}

		o := opt.fun(*opt.value)
		opts = append(opts, o)
	}

	return opts
}
