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

package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"fillmore-labs.com/scopeguard/internal/config"
	"fillmore-labs.com/scopeguard/internal/run"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/guidance"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/mcputil"
)

// AddScopeTool registers the "scope" tool.
func AddScopeTool(server *mcp.Server, s *mcputil.ServerState) {
	tool := &mcp.Tool{
		Name:  guidance.ScopeToolName,
		Title: "Scope",
		Description: "Apply scope-tightening fixes to Go source files. " +
			"Use safety filter to target specific tiers (see help topics 'unsafe' and 'breaking').",
	}

	overrides := CommonSchemaOverrides

	c := &ScopeContext{state: s}
	handler := mcputil.WrapTool(tool, c.Scope, overrides)

	server.AddTool(tool, handler)
}

// ScopeContext encapsulates the runtime state required for scope-tightening operations.
type ScopeContext struct {
	state *mcputil.ServerState
}

// ScopeArgs are the input arguments of the [Scope] tool.
type ScopeArgs struct {
	engine.Args
	Safety *engine.SafetyTiers `json:"safety,omitempty" jsonschema:"Filter returned diagnostics by safety tier: safe, unsafe, breaking. Omit to include all tiers. Fix generation is independent of this filter; every fix is assigned a tier, but this filter only narrows which ones appear in the response."`
	Limit  *int                `json:"limit,omitempty"  jsonschema:"Maximum number of preview/unapplied edits to return. Omit for the default (50). Applied edits are always returned. Use summary.total for the full count when results are truncated."`
	Mode   engine.ProcessMode  `json:"mode,omitzero"    jsonschema:"Processing mode. Omit (default): preview every fix as a diff without writing. 'apply': apply fixes whose IDs are listed in the apply field; all other fixes are still diff-rendered. 'apply_safe': apply every safe fix in one shot; unsafe and breaking fixes are still diff-rendered, never written. To apply unsafe or breaking fixes use 'apply' with explicit IDs (see help topics 'unsafe' and 'breaking')."`
	Apply  []engine.EditID     `json:"apply,omitempty"  jsonschema:"Edit IDs to apply. Required and only honored when mode='apply'. Obtain IDs from a prior preview call."`
}

// ScopeResult is the result of the [Scope] tool.
type ScopeResult struct {
	Message  string            `json:"message"            jsonschema:"Summary of what was found or applied"`
	Summary  engine.Counts     `json:"summary"            jsonschema:"Aggregate counts across all fixes (pre-truncation, pre-filter)"`
	Edits    []ScopeEdit       `json:"edits,omitempty"    jsonschema:"Edits previewed or applied (preview edits may be truncated; see summary.total)"`
	NextStep guidance.NextStep `json:"next_step,omitzero" jsonschema:"Recommended next action"`
}

// ScopeEdit describes a single scope-tightening edit.
type ScopeEdit struct {
	Message string `json:"message"           jsonschema:"Description of the scope issue"`
	Reason  string `json:"reason,omitzero"   jsonschema:"Why this fix is unsafe or breaking (empty when safe)"`
	DiffURI string `json:"diff_uri,omitzero" jsonschema:"URI of the embedded diff resource"`
	Diff    string `json:"diff,omitempty"    jsonschema:"Unified diff (always present in preview; present for unapplied edits in apply mode)"`
	engine.Position
	ID      engine.EditID `json:"id,omitzero"      jsonschema:"Stable identifier for this edit; pass to apply"`
	Safety  config.Safety `json:"safety"           jsonschema:"Safety tier: safe, unsafe, or breaking"`
	Applied bool          `json:"applied,omitzero" jsonschema:"Whether the edit was written to disk"`
}

// Scope is an MCP tool to tighten variable scopes.
//
// Three modes are supported via the Mode field:
//   - "preview" (default): returns every fix as a diff without writing anything.
//   - "apply": applies only the fixes whose IDs appear in the Apply field; all
//     other fixes are still diff-rendered.
//   - "apply_safe": applies every safe fix in one shot; unsafe and breaking
//     fixes are still diff-rendered and never written.
func (c ScopeContext) Scope(ctx context.Context, req *mcp.CallToolRequest, args ScopeArgs) (*ScopeResult, []mcp.Content, error) {
	filter := args.Safety.Filter()
	r := &run.Options{
		Analyzers: config.ScopeAnalyzer,
		Behaviors: config.CombineDeclarations,
		Functions: args.Functions,
		MaxLines:  maxLines,
		Filters:   config.NewFilters(filter, config.All), // We want all fixes, either to apply or preview
	}

	a := r.Analyzer()

	graph, err := engine.AnalyzePackages(ctx, args.Args, a)
	if err != nil {
		return nil, nil, err
	}

	allDiags := engine.AllDiagnostics(graph)

	summary := engine.Facts{Mode: args.Mode, Filter: filter}

	processed, err := engine.Process(allDiags, args.Mode, args.Apply, &summary)
	if err != nil {
		return nil, nil, err
	}

	limit := resolveLimit(args.Limit)
	prealloc := min(limit, len(processed))
	builder := c.state.NewResultBuilder(prealloc)

	edits := make([]ScopeEdit, 0, prealloc)

	for i, p := range processed {
		if i >= limit {
			break
		}

		diff, diffID := builder.EmbedDiff(p.Diff, p.ID)

		edits = append(edits, ScopeEdit{
			ID:       p.ID,
			DiffURI:  diffID,
			Message:  p.Message,
			Safety:   p.Info.Safety,
			Reason:   p.Info.Reason,
			Diff:     diff,
			Applied:  p.Applied,
			Position: p.Position,
		})
	}
	summary.Dropped = len(processed) - len(edits)

	g := guidance.ScopeContext{}
	out := &ScopeResult{
		Message:  guidance.ScopeMessage(summary, g),
		Summary:  summary.Summarize(),
		Edits:    edits,
		NextStep: guidance.ScopeRules.Dispatch(summary, g),
	}

	content := builder.ExtraContent(req.Session)

	return out, content, err
}
