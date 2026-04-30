// Copyright 2026 Oliver Eikemeier. All Rights Reserved.
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

package guidance

import (
	"fmt"
	"strings"

	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
)

// FilterPhrase returns the active safety-filter description like "(safety filters: safe)",
// or "" when no filter is active.
func FilterPhrase(f engine.Facts) string {
	if f.Filter.Default() {
		return ""
	}

	return fmt.Sprintf("(safety filters: %s)", strings.Join(f.Filter.Tiers(), ", "))
}

// TruncationPhrase returns "N not shown (<helpRef>)" when items were dropped
// due to a limit, otherwise "". helpTopic is passed to [HelpRef].
func TruncationPhrase(f engine.Facts, helpTopic string) string {
	if f.Dropped == 0 {
		return ""
	}

	return fmt.Sprintf("%d not shown (%s)", f.Dropped, HelpRef(helpTopic))
}

// HelpRef formats a reference to a named help topic: "see help topic 'X'".
func HelpRef(topic string) string {
	return fmt.Sprintf("see help topic '%s'", topic)
}

// Join concatenates non-empty strings with sep, silently dropping blank elements.
func Join(sep string, elems ...string) string {
	switch len(elems) {
	case 0:
		return ""
	case 1:
		return elems[0]
	}

	var size int

	first := true

	for _, elem := range elems {
		if elem == "" {
			continue
		}

		if first {
			first = false
		} else {
			size += len(sep)
		}
		size += len(elem)
	}

	var b strings.Builder
	b.Grow(size)

	first = true

	for _, elem := range elems {
		if elem == "" {
			continue
		}

		if first {
			first = false
		} else {
			b.WriteString(sep)
		}

		b.WriteString(elem)
	}

	return b.String()
}
