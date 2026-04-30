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

// ShadowContext carries the result context for shadow rule evaluation.
type ShadowContext struct{}

// ShadowMessage returns a human-readable summary for the shadow result.
func ShadowMessage(f engine.Facts, _ ShadowContext) string {
	if f.Total == 0 {
		return "No issues found"
	}

	return Join("; ",
		fmt.Sprintf("%d issue(s) found", f.Total),
		TruncationPhrase(f, "limits"),
	)
}

// ShadowRules defines next-step recommendations for shadowed variable renames.
var ShadowRules = Rules[ShadowContext]{
	// No results (apply with nothing to rename, or preview with no issues).
	func(f engine.Facts, _ ShadowContext) (NextStep, bool) {
		if f.Total != 0 {
			return NextStep{}, false
		}

		return DoneStep("No shadow issues found."), true
	},

	// Apply mode: confirm with "analyze".
	func(f engine.Facts, _ ShadowContext) (NextStep, bool) {
		if f.Mode == engine.ProcessPreview {
			return NextStep{}, false
		}

		return NextStep{NextTool: AnalyzeToolName, Recommendation: "Re-run analyze to confirm renames were applied correctly."}, true
	},

	// Preview with results; guide toward building a renames map.
	func(f engine.Facts, _ ShadowContext) (NextStep, bool) {
		var rec string

		if f.Dropped != 0 {
			rec = fmt.Sprintf("Build a renames map and call shadow to apply the %d shown rename(s); %s.", f.Total-f.Dropped, HelpRef("naming"))
		} else {
			rec = fmt.Sprintf("Build a renames map and call shadow to apply %d rename(s); %s.", f.Total, HelpRef("naming"))
		}

		return NextStep{NextTool: ShadowToolName, Recommendation: rec}, true
	},
}
