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
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func clientSession(tb testing.TB, s *mcp.Server) (*mcp.ClientSession, func()) {
	tb.Helper()

	in, out := mcp.NewInMemoryTransports()

	session, err := s.Connect(tb.Context(), in, nil)
	if err != nil {
		tb.Fatal("Can't create server session", err)
	}

	log := testLogger(tb, slog.LevelDebug)
	client := mcp.NewClient(&mcp.Implementation{}, &mcp.ClientOptions{Logger: log})

	cs, err := client.Connect(tb.Context(), out, nil)
	if err != nil {
		tb.Fatal("Can't create client session", err)
	}

	cleanup := func() {
		if err := cs.Close(); err != nil {
			tb.Error("failed to close the client connection", err)
		}

		if err := session.Close(); err != nil {
			tb.Error("failed to close the server connection", err)
		}
	}

	return cs, cleanup
}

func textResult(res *mcp.CallToolResult) string {
	if len(res.Content) > 0 {
		if tc, ok := res.Content[0].(*mcp.TextContent); ok {
			return tc.Text
		}
	}

	return ""
}

func resultAs[T any](res *mcp.CallToolResult) (string, T, error) {
	msg := textResult(res)

	var to T

	if res.IsError {
		return msg, to, errors.New(msg)
	}

	data, err := json.Marshal(res.StructuredContent)
	if err == nil {
		err = json.Unmarshal(data, &to)
	}

	return msg, to, err
}

func testLogger(tb testing.TB, level slog.Leveler) *slog.Logger {
	tb.Helper()

	handler := slog.NewTextHandler(tb.Output(), &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler)
}

func canReadResource(tb testing.TB, cs *mcp.ClientSession, diffURI, want string) {
	tb.Helper()

	res, err := cs.ReadResource(tb.Context(), &mcp.ReadResourceParams{URI: diffURI})
	if err != nil {
		tb.Fatalf("Can't read resource %s: %v", diffURI, err)
	}

	if len(res.Contents) != 1 {
		tb.Fatalf("Expected one resource, got %d", len(res.Contents))
	}

	if got := res.Contents[0].Text; got != want {
		tb.Errorf("Expected %q, got %q", want, got)
	}
}
