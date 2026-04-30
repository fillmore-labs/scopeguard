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
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/go/analysis/analysistest"

	"fillmore-labs.com/scopeguard/internal/typeutil"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/guidance"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/server"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/tools"
)

func TestAnalyze(t *testing.T) {
	t.Parallel()

	log := testLogger(t, slog.LevelDebug)

	testdata := analysistest.TestData()
	s := server.NewMCPServer(log, true)

	tests := [...]struct {
		name      string
		functions []typeutil.LocalFuncName
		wantErr   bool
		wantCount int
	}{
		{
			name:      "all",
			wantCount: 6,
		},
		{
			name:      "filtered",
			functions: []typeutil.LocalFuncName{{Name: "MoveToBlock"}},
			wantCount: 1,
		},
		{
			name:      "method",
			functions: []typeutil.LocalFuncName{{Receiver: "example", Name: "Method"}},
			wantCount: 1,
		},
		{
			name:      "unmatched",
			functions: []typeutil.LocalFuncName{{Name: "NonExistent"}},
			wantErr:   true,
		},
		{
			name:    "broken",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := filepath.Join(testdata, tt.name)

			cs, cleanup := clientSession(t, s)
			defer cleanup()

			res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{
				Name: guidance.AnalyzeToolName,
				Arguments: tools.AnalyzeArgs{
					Args: engine.Args{
						Dir:       dir,
						Functions: tt.functions,
					},
				},
			})
			if err != nil {
				t.Fatalf("Analyze() returned error: %v", err)
			}

			if tt.wantErr != res.IsError {
				if res.IsError && len(res.Content) > 0 {
					if tc, ok := res.Content[0].(*mcp.TextContent); ok {
						t.Fatalf("Analyze() error message: %s", tc.Text)
					}
				}

				t.Fatalf("Analyze() error = %t, want %t", res.IsError, tt.wantErr)
			}

			if res.IsError {
				return
			}

			_, sc, err := resultAs[*tools.AnalyzeResult](res)
			if err != nil {
				t.Fatalf("Analyze() returned invalid structured content: %v", err)
			}

			if got := len(sc.Issues); got != tt.wantCount {
				t.Fatalf("Analyze() returned %d issues, want %d", got, tt.wantCount)
			}

			for i, d := range sc.Issues {
				if d.Message == "" {
					t.Errorf("issue[%d]: empty message", i)
				}

				if d.File == "" {
					t.Errorf("issue[%d]: empty file", i)
				}

				if d.Line == 0 {
					t.Errorf("issue[%d]: missing line number", i)
				}

				if d.Category == 0 {
					t.Errorf("issue[%d]: empty category", i)
				}
			}
		})
	}
}
