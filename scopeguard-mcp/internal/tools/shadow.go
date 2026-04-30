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
	"fillmore-labs.com/scopeguard/internal/typeutil"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/guidance"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/mcputil"
)

// AddShadowTool registers the "shadow" tool.
func AddShadowTool(server *mcp.Server, s *mcputil.ServerState) {
	tool := &mcp.Tool{
		Name:        guidance.ShadowToolName,
		Title:       "Shadow",
		Description: "Rename shadowed variables in Go source files.",
	}

	overrides := CommonSchemaOverrides

	c := &ShadowContext{state: s}
	handler := mcputil.WrapTool(tool, c.Shadow, overrides)

	server.AddTool(tool, handler)
}

// ShadowContext is a context structure used for managing operations related to renaming shadowed variables.
type ShadowContext struct {
	state *mcputil.ServerState
}

// ShadowArgs are the input arguments of the [Shadow] tool.
//
// Note: this tool uses the `write` boolean to commit changes, distinct from the
// `apply` slice on the scope tool which lists edit IDs.
type ShadowArgs struct {
	engine.Args
	Limit   *int               `json:"limit,omitempty"   jsonschema:"Maximum number of edits to return. Omit for the default (50). Applied edits are always returned. Use summary.total for the full count when results are truncated."`
	Renames typeutil.RenameMap `json:"renames,omitempty" jsonschema:"Strongly recommended. Map of function names to renames. For each function, map the original variable name to an ordered list of replacement names: the n-th shadowing occurrence is renamed to the n-th name in the list. See help topic 'naming'."`
	Write   bool               `json:"write,omitzero"    jsonschema:"Write the renames to disk. Omit (default) for preview: returns unified diffs without writing. Set true to commit the renames into the source files."`
}

// ShadowResult is the result of the [Shadow] tool.
type ShadowResult struct {
	Message  string            `json:"message"            jsonschema:"Summary of what was found or applied"`
	Summary  engine.Counts     `json:"summary"            jsonschema:"Aggregate counts across all renames (pre-truncation, pre-filter)"`
	Edits    []ShadowEdit      `json:"edits,omitempty"    jsonschema:"Renames applied or previewed (may be truncated; see summary.total)"`
	NextStep guidance.NextStep `json:"next_step,omitzero" jsonschema:"Recommended next action"`
}

// ShadowEdit describes a single rename of a shadowed variable.
type ShadowEdit struct {
	Edit    string `json:"edit"              jsonschema:"Description of the rename"`
	DiffURI string `json:"diff_uri,omitzero" jsonschema:"URI of the embedded diff resource"`
	Diff    string `json:"diff,omitempty"    jsonschema:"Unified diff (preview mode only)"`
	engine.Position
}

// Shadow is an MCP tool to rename shadowed variables.
func (c ShadowContext) Shadow(ctx context.Context, req *mcp.CallToolRequest, args ShadowArgs) (*ShadowResult, []mcp.Content, error) {
	filter := config.All // renames are safe

	behaviors := config.FirstUseOnly | config.RenameVariables
	processMode := engine.ProcessPreview

	if args.Write {
		behaviors = config.FirstUseOnly
		processMode = engine.ProcessApplySafe
	}

	r := &run.Options{
		Analyzers: config.ShadowAnalyzer,
		Behaviors: behaviors,
		Functions: args.Functions,
		Renames:   args.Renames,
		MaxLines:  maxLines,
		Filters:   config.NewFilters(filter, config.All), // We want all fixes, either to apply or preview
	}

	a := r.Analyzer()

	graph, err := engine.AnalyzePackages(ctx, args.Args, a)
	if err != nil {
		return nil, nil, err
	}

	allDiags := engine.AllDiagnostics(graph)

	summary := engine.Facts{Mode: processMode, Filter: filter}

	processed, err := engine.Process(allDiags, processMode, nil, &summary)
	if err != nil {
		return nil, nil, err
	}

	limit := resolveLimit(args.Limit)
	prealloc := min(limit, len(processed))
	builder := c.state.NewResultBuilder(prealloc)

	edits := make([]ShadowEdit, 0, prealloc)

	for i, p := range processed {
		if i >= limit {
			break
		}

		diff, diffID := builder.EmbedDiff(p.Diff, p.ID)

		edits = append(edits, ShadowEdit{
			Edit:     p.Edit,
			Diff:     diff,
			DiffURI:  diffID,
			Position: p.Position,
		})
	}
	summary.Dropped = len(processed) - len(edits)

	g := guidance.ShadowContext{}
	out := &ShadowResult{
		Message:  guidance.ShadowMessage(summary, g),
		Summary:  summary.Summarize(),
		Edits:    edits,
		NextStep: guidance.ShadowRules.Dispatch(summary, g),
	}

	extracontent := builder.ExtraContent(req.Session)

	return out, extracontent, nil
}
