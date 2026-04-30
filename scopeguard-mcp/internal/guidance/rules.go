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

	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
)

// Rules is a declarative dispatch table.
// Each rule returns true when it should fire, along with the [NextStep] to surface.
//
// Rules are evaluated in order; the first match wins. The last rule must be an
// unconditional catch-all so [Rules.Dispatch] never falls off the end.
//
// C carries per-call context so additional context can be added besides [engine.Facts].
// Empty structs have zero size, so passing this by value to every rule is free at runtime.
type Rules[C any] []func(engine.Facts, C) (NextStep, bool)

// Dispatch evaluates rules in order and returns the NextStep of the first matching rule.
// If no rule matches, it returns a "done" step describing the gap; with a correctly
// built rule table this branch is unreachable because the final rule is a catch-all.
func (r Rules[C]) Dispatch(f engine.Facts, context C) NextStep {
	for _, rule := range r {
		if nextstep, ok := rule(f, context); ok {
			return nextstep
		}
	}

	return DoneStep("No applicable next step (no rule matched).")
}

// DoneStep creates a NextStep with the "done" action, indicating no further tool calls are needed.
func DoneStep(format string, args ...any) NextStep {
	return NextStep{NextTool: DoneToolName, Recommendation: fmt.Sprintf(format, args...)}
}
