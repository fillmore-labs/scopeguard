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

// NextStep guides the caller toward the next action in the workflow.
type NextStep struct {
	NextTool       string         `json:"next_tool"      jsonschema:"Next tool to call: 'scope', 'shadow', 'analyze', or 'done'"`
	Args           map[string]any `json:"args,omitempty" jsonschema:"Suggested arguments for the next call (omitted when action is 'done' or no specific args are suggested)"`
	Recommendation string         `json:"recommendation" jsonschema:"Short description of what to do and why"`
}

const (
	// AnalyzeToolName is the name of the "analyze" tool used for reporting issues.
	AnalyzeToolName = "analyze"

	// HelpToolName is the name for the help tool, used to retrieve reference material based on topics.
	HelpToolName = "help"

	// ScopeToolName is the name of the tool used to apply scope-tightening fixes to Go source code.
	ScopeToolName = "scope"

	// ShadowToolName is the name of the tool used for finding and renaming shadowed variables.
	ShadowToolName = "shadow"

	// DoneToolName is the action value returned when no further tool calls are needed.
	DoneToolName = "done"
)
