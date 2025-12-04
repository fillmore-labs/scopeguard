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

package analyze

import "fillmore-labs.com/scopeguard/analyzer/level"

// Options represent configuration options for the scopeguard analyzer.
type Options struct {
	// Generated specifies whether to include analysis of generated files.
	Generated bool

	// MaxLines specifies the maximum number of lines a declaration can span to be considered for moving
	// into control flow initializers.
	MaxLines int

	// ScopeLevel determines which scope checks are enabled.
	ScopeLevel level.Scope

	// ShadowLevel determines which shadow checks are enabled.
	ShadowLevel level.Shadow

	// NestedAssign determines which nested assign checks are enabled.
	NestedAssign level.NestedAssign
}

// DefaultOptions initializes and returns a new Options instance with default values.
func DefaultOptions() *Options {
	o := &Options{
		Generated:    false,
		MaxLines:     -1,
		ScopeLevel:   level.ScopeFull,
		ShadowLevel:  level.ShadowFull,
		NestedAssign: level.NestedFull,
	}

	return o
}
