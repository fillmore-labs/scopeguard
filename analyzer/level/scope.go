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

package level

import (
	"fmt"
	"strings"
)

// Scope specifies the scope analysis level.
type Scope uint8

const (
	// ScopeFull enables all scope checks.
	ScopeFull Scope = iota

	// ScopeConservative specifies to only permit moves that don't cross code with potential side effects.
	ScopeConservative

	// ScopeOff disables all scope checks.
	ScopeOff
)

// MarshalText implements [encoding.TextMarshaler].
func (o Scope) MarshalText() ([]byte, error) {
	switch o {
	case ScopeFull:
		return []byte("full"), nil

	case ScopeConservative:
		return []byte("conservative"), nil

	case ScopeOff:
		return []byte("off"), nil

	default:
		return nil, fmt.Errorf("unknown scope level %d", o)
	}
}

// UnmarshalText implements [encoding.TextUnmarshaler].
func (o *Scope) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	case "", "true", "on", "full":
		*o = ScopeFull

	case "conservative":
		*o = ScopeConservative

	case "off", "false":
		*o = ScopeOff

	default:
		return fmt.Errorf("unknown scope level %q", string(text))
	}

	return nil
}
