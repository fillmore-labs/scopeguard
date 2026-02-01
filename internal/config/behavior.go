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

package config

// DefaultBehavior returns the default behavior settings.
func DefaultBehavior() Behavior { return FirstUseOnly | CombineDeclarations | RenameVariables }

// Behavior is representing a set of configurable binary flags for behavior control.
type Behavior uint8

const (
	// IncludeGenerated specifies whether to include analysis of generated files.
	IncludeGenerated Behavior = 1 << iota

	// FirstUseOnly specifies whether only the first use of a variable after being shadowed should be reported.
	FirstUseOnly

	// CombineDeclarations determines whether to combine declarations when moving to init statements.
	CombineDeclarations

	// Conservative indicates that moves should be conservative.
	Conservative

	// RenameVariables indicates that shadowed variables should be renamed.
	RenameVariables
)

var _behavior = [...]string{
	"generated",
	"first-only",
	"combine",
	"conservative",
	"rename",
}

func (b Behavior) String() string { return toString(b, _behavior[:], "none") }

// Set adjusts the bitmask by enabling or disabling the specified option.
func (b *Behavior) Set(flag Behavior, value bool) { set(b, flag, value) }

// Enabled checks if the specified option is enabled in the current bitmask.
func (b Behavior) Enabled(flag Behavior) bool { return b&flag != 0 }
