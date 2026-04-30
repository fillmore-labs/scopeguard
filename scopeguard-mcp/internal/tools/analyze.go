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
	"iter"
	"slices"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"fillmore-labs.com/scopeguard/internal/config"
	"fillmore-labs.com/scopeguard/internal/run"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/guidance"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/mcputil"
)

// AddAnalyzeTool registers the "analyze" tool.
func AddAnalyzeTool(server *mcp.Server) {
	tool := &mcp.Tool{
		Name:        guidance.AnalyzeToolName,
		Title:       "Analyze",
		Description: "Report scope and shadow issues in Go code. Diagnostics only, no file changes. Start here.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}

	overrides := CommonSchemaOverrides

	c := AnalyzeContext{}
	handler := mcputil.WrapTool(tool, c.Analyze, overrides)

	server.AddTool(tool, handler)
}

// AnalyzeContext represents the context used for analyzing scope and shadow issues in Go code.
type AnalyzeContext struct{}

// AnalyzeArgs are the input arguments of the [Analyze] tool.
type AnalyzeArgs struct {
	engine.Args
	Safety *engine.SafetyTiers `json:"safety,omitempty" jsonschema:"Filter returned issues by safety tier: safe, unsafe, breaking. Omit to return all tiers."`
	Limit  *int                `json:"limit,omitempty"  jsonschema:"Maximum number of issues to return. Omit for the default (50). Use the summary fields for totals when results are truncated."`
}

// AnalyzeResult is the result of the [Analyze] tool.
type AnalyzeResult struct {
	Message  string            `json:"message"            jsonschema:"Summary of the analysis results."`
	Summary  engine.Counts     `json:"summary"            jsonschema:"Aggregate counts across all matching issues (pre-truncation)."`
	Issues   []AnalyzeIssue    `json:"issues,omitempty"   jsonschema:"Issues found (may be truncated; check summary.total for the full count)."`
	NextStep guidance.NextStep `json:"next_step,omitzero" jsonschema:"Recommended next action"`
}

// AnalyzeIssue is a single issue found during analysis.
type AnalyzeIssue struct {
	Message  string       `json:"message"          jsonschema:"Description of the issue"`
	Category engine.Issue `json:"category"         jsonschema:"Issue category: scope or shadow"`
	Reason   string       `json:"reason,omitempty" jsonschema:"Why the suggested fix would be unsafe or breaking; empty when the fix is safe"`
	engine.Position
	Safety config.Safety `json:"safety" jsonschema:"Safety tier of the suggested fix: safe, unsafe, or breaking"`
}

// Compare compares two AnalyzeIssue instances lexicographically by file path and numerically by line number.
func (a AnalyzeIssue) Compare(b AnalyzeIssue) int {
	return a.Position.Compare(b.Position)
}

// Analyze is an MCP tool that reports all scope and shadow issues.
func (AnalyzeContext) Analyze(ctx context.Context, _ *mcp.CallToolRequest, args AnalyzeArgs) (*AnalyzeResult, []mcp.Content, error) {
	filter := args.Safety.Filter()

	r := &run.Options{
		Analyzers: config.ScopeAnalyzer | config.ShadowAnalyzer,
		Behaviors: config.FirstUseOnly | config.CombineDeclarations,
		Functions: args.Functions,
		MaxLines:  maxLines,
		Filters:   config.NewFilters(filter, config.Nothing), // No fixes for analyze
	}

	a := r.Analyzer()

	graph, err := engine.AnalyzePackages(ctx, args.Args, a)
	if err != nil {
		return nil, nil, err
	}

	allDiags := engine.AllDiagnostics(graph)

	summary := engine.Facts{
		Mode:   engine.ProcessPreview,
		Filter: filter,
		Counts: engine.Counts{
			ByCategory: make(map[engine.Issue]int),
			BySafety:   make(map[config.Safety]int),
		},
	}

	limit := resolveLimit(args.Limit)
	issues := collectAnalyzeIssues(allDiags, limit, &summary)

	c := guidance.AnalyzeContext{}
	msg := guidance.AnalyzeMessage(summary, c)
	out := &AnalyzeResult{
		Message:  msg,
		Summary:  summary.Summarize(),
		Issues:   issues,
		NextStep: guidance.AnalyzeRules.Dispatch(summary, c),
	}

	return out, nil, nil
}

// collectAnalyzeIssues projects diagnostics into AnalyzeIssue records, tracking all and limiting returned.
// All diagnostics contribute to summary counts regardless of the safety filter or limit.
func collectAnalyzeIssues(diagnostics iter.Seq[engine.AnalyzerDiagnostic], limit int, summary *engine.Facts) []AnalyzeIssue {
	var issues []AnalyzeIssue

	var total int

	for diag := range diagnostics {
		info := diag.Info
		summary.AddDiagnostic(info)

		total++

		if len(issues) < limit {
			issues = append(issues, AnalyzeIssue{
				Message:  diag.Message,
				Category: info.Issue,
				Safety:   info.Safety,
				Reason:   info.Reason,
				Position: diag.Position,
			})
		}
	}

	summary.Total = total
	summary.Dropped = total - len(issues)

	slices.SortStableFunc(issues, AnalyzeIssue.Compare)

	return issues
}
