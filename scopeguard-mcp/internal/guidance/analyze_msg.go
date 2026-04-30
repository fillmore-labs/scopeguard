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
	"maps"
	"slices"
	"strings"

	"fillmore-labs.com/scopeguard/internal/config"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
)

// AnalyzeContext carries per-call context for rule evaluation.
type AnalyzeContext struct{}

// AnalyzeMessage generates a summary string of analysis results based on issue details, summary facts, and safety filters.
func AnalyzeMessage(f engine.Facts, _ AnalyzeContext) string {
	if f.Total == 0 {
		if !f.Filter.Default() {
			return "No matching issues found (safety filter active)"
		}

		return "No issues found"
	}

	parts := make([]string, 0, len(f.ByCategory))
	for _, class := range slices.Sorted(maps.Keys(f.ByCategory)) {
		parts = append(parts, fmt.Sprintf("%d %s", f.ByCategory[class], class))
	}

	var mainMsg string
	if len(parts) == 0 {
		mainMsg = fmt.Sprintf("%d issue(s) found", f.Total)
	} else {
		mainMsg = strings.Join(parts, ", ") + " issue(s) found"
	}

	if msg := FilterPhrase(f); msg != "" {
		mainMsg += " " + msg
	}

	var breakingPart string
	if breaking := f.BySafety[config.Breaking]; breaking > 0 {
		breakingPart = fmt.Sprintf("%d issue(s) have compilation-breaking fixes (%s)", breaking, HelpRef("breaking"))
	}

	return Join("; ", mainMsg, breakingPart, TruncationPhrase(f, "limits"))
}

// strategyThreshold defines the threshold value used to determine the application of a "too much" strategy.
const strategyThreshold = 200

// AnalyzeRules defines next-step recommendations for analysis results.
var AnalyzeRules = Rules[AnalyzeContext]{
	// No issues.
	func(f engine.Facts, _ AnalyzeContext) (NextStep, bool) {
		if f.Total != 0 {
			return NextStep{}, false
		}

		return DoneStep("No issues found."), true
	},

	// Overload: try fixing safe scope issues blindly.
	func(f engine.Facts, _ AnalyzeContext) (NextStep, bool) {
		if f.BySafety[config.Safe] < strategyThreshold {
			return NextStep{}, false
		}

		return NextStep{
			NextTool:       ScopeToolName,
			Args:           map[string]any{"safety": []string{"safe"}, "mode": "apply_safe"},
			Recommendation: fmt.Sprintf("%d safe issues found: consider a blind safe-apply (%s).", f.BySafety[config.Safe], HelpRef("strategy")),
		}, true
	},

	// Overload: steer toward scoped iteration for very large codebases.
	func(f engine.Facts, _ AnalyzeContext) (NextStep, bool) {
		if f.Total < strategyThreshold {
			return NextStep{}, false
		}

		return NextStep{
			NextTool:       ScopeToolName,
			Recommendation: fmt.Sprintf("%d issues found: consider function-scoped iteration (%s).", f.Total, HelpRef("strategy")),
		}, true
	},

	// Scope issues first.
	func(f engine.Facts, _ AnalyzeContext) (NextStep, bool) {
		if f.ByCategory[engine.IssueScope] == 0 {
			return NextStep{}, false
		}

		return NextStep{
			NextTool:       ScopeToolName,
			Recommendation: fmt.Sprintf("Preview %d scope fix(es) to get IDs and diffs.", f.ByCategory[engine.IssueScope]),
		}, true
	},

	// Shadow issues.
	func(f engine.Facts, _ AnalyzeContext) (NextStep, bool) {
		if f.ByCategory[engine.IssueShadow] == 0 {
			return NextStep{}, false
		}

		return NextStep{
			NextTool:       ShadowToolName,
			Recommendation: fmt.Sprintf("Preview %d shadow rename(s) (omit 'write' for preview)", f.ByCategory[engine.IssueShadow]),
		}, true
	},

	// Catch-all: issues exist but none have an automated fix path
	// (e.g. nested-assignment diagnostics).
	func(f engine.Facts, _ AnalyzeContext) (NextStep, bool) {
		return DoneStep("%d issue(s) require manual review.", f.Total), true
	},
}
