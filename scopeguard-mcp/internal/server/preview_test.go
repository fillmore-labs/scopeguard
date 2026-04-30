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
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/go/analysis/analysistest"

	"fillmore-labs.com/scopeguard/internal/typeutil"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/guidance"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/server"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/tools"
)

func TestScopePreview(t *testing.T) {
	t.Parallel()

	log := testLogger(t, slog.LevelDebug)

	testdata := analysistest.TestData()
	s := server.NewMCPServer(log, true)

	dir := filepath.Join(testdata, "all")

	cs, cleanup := clientSession(t, s)
	defer cleanup()

	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "scope",
		Arguments: tools.ScopeArgs{Args: engine.Args{Dir: dir}},
	})
	if err != nil {
		t.Fatalf("Scope(preview) returned error: %v", err)
	}

	_, sc, err := resultAs[tools.ScopeResult](res)
	if err != nil {
		t.Fatalf("Scope(preview) returned invalid structured content: %v", err)
	}

	if len(sc.Edits) == 0 {
		t.Fatal("Scope(preview) returned no edits")
	}

	for i, e := range sc.Edits {
		if e.Message == "" {
			t.Errorf("edit[%d]: empty message", i)
		}

		if e.ID == 0 {
			t.Errorf("edit[%d]: empty ID", i)
		}

		var diff string

		switch {
		case e.DiffURI != "":
			var resource *mcp.ResourceContents

			for _, c := range res.Content {
				er, ok := c.(*mcp.EmbeddedResource)
				if !ok {
					continue
				}

				if er.Resource.URI != e.DiffURI {
					continue
				}

				resource = er.Resource

				break
			}

			if resource == nil {
				t.Fatalf("Resource %s missing", e.DiffURI)
			}

			diff = resource.Text

			canReadResource(t, cs, e.DiffURI, resource.Text)

			if !strings.Contains(resource.Text, "@@") {
				t.Errorf("edit[%d]: diff missing hunk header:\n%s", i, resource.Text)
			}

		default:
			diff = e.Diff
		}

		if e.ID == 0 {
			t.Errorf("edit[%d]: empty ID", i)
		}

		if !strings.Contains(diff, "@@") {
			t.Errorf("edit[%d]: diff missing hunk header:\n%s", i, diff)
		}
	}
}

func TestShadowPreview(t *testing.T) {
	t.Parallel()

	log := testLogger(t, slog.LevelDebug)

	testdata := analysistest.TestData()
	s := server.NewMCPServer(log, true)

	dir := filepath.Join(testdata, "rename")

	cs, cleanup := clientSession(t, s)
	defer cleanup()

	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{
		Name: guidance.ShadowToolName,
		Arguments: tools.ShadowArgs{
			Args:    engine.Args{Dir: dir},
			Renames: typeutil.RenameMap{{Name: "ShadowedErr"}: {"err": {"outerErr"}}},
		},
	})
	if err != nil {
		t.Fatalf("Shadow(preview) returned error: %v", err)
	}

	_, sc, err := resultAs[tools.ShadowResult](res)
	if err != nil {
		t.Fatalf("Shadow(preview) returned invalid structured content: %v", err)
	}

	if got := len(sc.Edits); got != 2 {
		t.Fatalf("Shadow(preview) returned %d edits, want 2", got)
	}

	e := sc.Edits[0]

	if got, want := e.Edit, "Rename 'err' to 'outerErr'"; got != want {
		t.Errorf("Message = %q, want %q", got, want)
	}

	if got, want := e.Function, "ShadowedErr"; got != want {
		t.Errorf("Function = %q, want %q", got, want)
	}

	if got, want := e.File, "example.go"; !strings.HasSuffix(got, want) {
		t.Errorf("File = %q, want suffix %q", got, want)
	}

	var diff string

	switch {
	case e.DiffURI != "":
		var resource *mcp.ResourceContents

		for _, c := range res.Content {
			er, ok := c.(*mcp.EmbeddedResource)
			if !ok {
				continue
			}

			if er.Resource.URI != e.DiffURI {
				continue
			}

			resource = er.Resource

			break
		}

		if resource == nil {
			t.Fatalf("Resource %s missing", e.DiffURI)
		}

		diff = resource.Text

		canReadResource(t, cs, e.DiffURI, diff)

	default:
		diff = e.Diff
	}

	if got, want := diff, "outerErr"; !strings.Contains(got, want) {
		t.Errorf("Diff does not contain %q:\n%s", want, got)
	}
}
