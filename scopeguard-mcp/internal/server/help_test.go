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
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/help"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/server"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/tools"
)

func TestHelpTool(t *testing.T) {
	t.Parallel()

	log := testLogger(t, slog.LevelDebug)

	s := server.NewMCPServer(log, true)

	tests := [...]struct {
		name       string
		topic      tools.HelpTopic
		wantSubstr string
		wantErr    bool
	}{
		{name: "naming", topic: "naming", wantSubstr: "Naming shadowed variables"},
		{name: "unsafe", topic: "unsafe", wantSubstr: "Unsafe scope fixes"},
		{name: "all", topic: "", wantSubstr: "Naming"},
		{name: "unknown topic", topic: "bogus", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cs, cleanup := clientSession(t, s)
			defer cleanup()

			res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{
				Name:      "help",
				Arguments: tools.HelpArgs{Topic: tt.topic},
			})
			if err != nil {
				t.Fatalf("CallTool(help) returned error: %v", err)
			}

			if tt.wantErr {
				if !res.IsError {
					t.Fatalf("CallTool(help topic=%q) IsError = false, want true", tt.topic)
				}

				if len(res.Content) == 0 {
					t.Fatalf("CallTool(help topic=%q) error result has no content", tt.topic)
				}

				tc, ok := res.Content[0].(*mcp.TextContent)
				if !ok {
					t.Fatalf("CallTool(help topic=%q) error content is %T, want *mcp.TextContent", tt.topic, res.Content[0])
				}

				for _, topic := range help.AllTopics() {
					if !strings.Contains(tc.Text, topic) {
						t.Errorf("error message missing topic %q: %q", topic, tc.Text)
					}
				}

				return
			}

			if res.IsError {
				t.Fatalf("CallTool(help topic=%q) IsError = true, want false", tt.topic)
			}

			tc, ok := res.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatalf("CallTool(help topic=%q) content is %T, want *mcp.TextContent", tt.topic, res.Content[0])
			}

			if !strings.Contains(tc.Text, tt.wantSubstr) {
				t.Errorf("Content missing %q:\n%s", tt.wantSubstr, tc.Text)
			}

			if tt.topic == "" {
				return
			}

			resource, err := cs.ReadResource(t.Context(), &mcp.ReadResourceParams{
				URI: tools.HelpPrefix + string(tt.topic),
			})
			if err != nil {
				t.Fatalf("ReadResource(help) returned error: %v", err)
			}

			if len(resource.Contents) != 1 {
				t.Fatalf("ReadResource(help) returned %d contents, want 1", len(resource.Contents))
			}

			if got, want := resource.Contents[0].Text, tc.Text; got != want {
				t.Errorf("ReadResource(help) returned %q, expected %q", got, want)
			}
		})
	}
}
