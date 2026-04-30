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

package server_test

import (
	"log/slog"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/guidance"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/server"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/tools"
)

func TestPathNotAbsolute(t *testing.T) {
	t.Parallel()

	log := testLogger(t, slog.LevelDebug)

	s := server.NewMCPServer(log, true)

	tests := [...]struct {
		name string
		args any
		tool string
	}{
		{
			name: "analyze",
			tool: guidance.AnalyzeToolName,
			args: tools.AnalyzeArgs{Args: engine.Args{Dir: "relative/path"}},
		},
		{
			name: "scope",
			tool: guidance.ScopeToolName,
			args: tools.ScopeArgs{Args: engine.Args{Dir: "relative/path"}},
		},
		{
			name: "shadow",
			tool: guidance.ShadowToolName,
			args: tools.ShadowArgs{Args: engine.Args{Dir: "relative/path"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cs, cleanup := clientSession(t, s)
			defer cleanup()

			params := &mcp.CallToolParams{
				Name:      tt.tool,
				Arguments: tt.args,
			}

			res, err := cs.CallTool(t.Context(), params)
			if err != nil {
				t.Fatalf("%s() returned error: %v", tt.tool, err)
			}

			if !res.IsError {
				msg := textResult(res)
				t.Fatalf("%s() should have returned an error for relative path (%q)", tt.tool, msg)
			}
		})
	}
}
