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

// NestedAssign specifies the nested assign level.
type NestedAssign uint8

const (
	// NestedFull enables all nested assign checks.
	NestedFull NestedAssign = iota

	// NestedOff disables all nested assign checks.
	NestedOff
)

// MarshalText implements [encoding.TextMarshaler].
func (o NestedAssign) MarshalText() ([]byte, error) {
	switch o {
	case NestedFull:
		return []byte("full"), nil

	case NestedOff:
		return []byte("off"), nil

	default:
		return nil, fmt.Errorf("unknown nested assign level %d", o)
	}
}

// UnmarshalText implements [encoding.TextUnmarshaler].
func (o *NestedAssign) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	case "", "true", "on", "full":
		*o = NestedFull

	case "off", "false":
		*o = NestedOff

	default:
		return fmt.Errorf("unknown nested assign level %q", string(text))
	}

	return nil
}
