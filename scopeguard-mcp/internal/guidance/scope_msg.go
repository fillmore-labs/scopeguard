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

	"fillmore-labs.com/scopeguard/internal/config"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
)

// ScopeContext carries edit-application context for scope rule evaluation.
type ScopeContext struct{}

// ScopeMessage formats a summary message for scope edits, reflecting applied fixes, safety filters, and truncation info.
func ScopeMessage(f engine.Facts, _ ScopeContext) string {
	if f.Total == 0 {
		return "No fixes available"
	}

	var mainMsg string

	switch f.Applied {
	case 0:
		mainMsg = fmt.Sprintf("%d fix(es) available (not applied)", f.Total)

	case f.Total:
		mainMsg = fmt.Sprintf("%d fix(es) applied", f.Applied)

	default:
		mainMsg = fmt.Sprintf("%d fix(es) applied; %d fix(es) available (not applied)", f.Applied, f.Total-f.Applied)
	}

	if msg := FilterPhrase(f); msg != "" {
		mainMsg += " " + msg
	}

	var breakingPart string

	if breaking := f.BySafety[config.Breaking]; breaking > 0 {
		breakingPart = fmt.Sprintf("%d breaking fix(es): treat diffs as scaffold (%s)", breaking, HelpRef("breaking"))
	}

	return Join("; ", mainMsg, breakingPart, TruncationPhrase(f, "limits"))
}

// ScopeRules defines next-step recommendations for scope-tightening fixes.
var ScopeRules = Rules[ScopeContext]{
	// No fixes available at all.
	func(f engine.Facts, _ ScopeContext) (NextStep, bool) {
		if f.Total != 0 {
			return NextStep{}, false
		}

		return DoneStep("No scope fixes available."), true
	},

	// After any "apply" mode: re-run "analyze" to confirm.
	func(f engine.Facts, _ ScopeContext) (NextStep, bool) {
		if f.Mode == engine.ProcessPreview {
			return NextStep{}, false
		}

		rec := "Re-run analyze to confirm fixes were applied correctly."
		if remaining := f.Total - f.Applied; remaining > 0 {
			rec = fmt.Sprintf(
				"Re-run analyze to confirm fixes were applied correctly; %d fix(es) remain.",
				remaining,
			)
		}

		return NextStep{NextTool: AnalyzeToolName, Recommendation: rec}, true
	},

	// Only safe fixes: apply_safe is the right shortcut (covers any truncated safe fixes too).
	func(f engine.Facts, _ ScopeContext) (NextStep, bool) {
		if f.BySafety[config.Safe] == 0 {
			return NextStep{}, false
		}

		safeTotal := f.BySafety[config.Safe]

		var rec string
		if f.Dropped != 0 {
			rec = fmt.Sprintf("Apply all %d safe fix(es) (including those not shown) with 'apply_safe', then re-run analyze.", safeTotal)
		} else {
			rec = fmt.Sprintf("Apply %d safe fix(es) with 'apply_safe', then re-run analyze.", safeTotal)
		}

		return NextStep{
			NextTool:       ScopeToolName,
			Args:           map[string]any{"safety": []string{"safe"}, "mode": "apply_safe"},
			Recommendation: rec,
		}, true
	},

	// Unsafe diffs: guide toward reviewing what's available.
	func(f engine.Facts, _ ScopeContext) (NextStep, bool) {
		if f.BySafety[config.Unsafe] == 0 {
			return NextStep{}, false
		}

		return NextStep{
			NextTool: ScopeToolName,
			Args:     map[string]any{"mode": "apply", "apply": []string{"... edit id list ..."}},
			Recommendation: fmt.Sprintf(
				"Review %d unsafe diff(s) before applying (%s).",
				f.BySafety[config.Unsafe], HelpRef("unsafe"),
			),
		}, true
	},

	// Only breaking left.
	func(f engine.Facts, _ ScopeContext) (NextStep, bool) {
		return NextStep{
			NextTool: ScopeToolName,
			Args:     map[string]any{"mode": "apply", "apply": []string{"... edit id list ..."}},
			Recommendation: fmt.Sprintf(
				"Review %d breaking diff(s) before applying (%s).",
				f.BySafety[config.Breaking], HelpRef("breaking"),
			),
		}, true
	},
}
