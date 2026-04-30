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
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/go/analysis/analysistest"

	"fillmore-labs.com/scopeguard/internal/config"
	"fillmore-labs.com/scopeguard/internal/typeutil"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/diff"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/guidance"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/server"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/tools"
)

func TestPatchScope(t *testing.T) {
	t.Parallel()

	log := testLogger(t, slog.LevelDebug)

	testdata := analysistest.TestData()
	s := server.NewMCPServer(log, true)

	tests := [...]struct {
		name    string
		srcDir  string
		args    tools.ScopeArgs
		wantErr bool
	}{
		{
			name:   "all",
			srcDir: "all",
			args:   tools.ScopeArgs{},
		},
		{
			name:   "filtered",
			srcDir: "filtered",
			args: tools.ScopeArgs{
				Args: engine.Args{
					Functions: []typeutil.LocalFuncName{{Name: "MoveToBlock"}},
				},
			},
		},
		{
			name:   "method",
			srcDir: "method",
			args: tools.ScopeArgs{
				Args: engine.Args{
					Functions: []typeutil.LocalFuncName{{Receiver: "example", Name: "Method"}},
				},
			},
		},
		{
			name:    "unmatched",
			srcDir:  "unmatched",
			args:    tools.ScopeArgs{Args: engine.Args{Functions: []typeutil.LocalFuncName{{Name: "NonExistent"}}}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srcDir := filepath.Join(testdata, tt.srcDir)

			dir := t.TempDir()
			if err := os.CopyFS(dir, os.DirFS(srcDir)); err != nil {
				t.Fatalf("copying %s to %s failed: %v", srcDir, dir, err)
			}

			cs, cleanup := clientSession(t, s)
			defer cleanup()

			// Step 1: preview to collect edit IDs.
			previewArgs := tt.args
			previewArgs.Dir = dir

			res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{
				Name:      "scope",
				Arguments: previewArgs,
			})
			if err != nil {
				t.Fatalf("scope(preview) returned error: %v", err)
			}

			if tt.wantErr {
				if !res.IsError {
					t.Fatalf("scope(preview) expected error but got none")
				}

				return
			}

			_, sc, err := resultAs[tools.ScopeResult](res)
			if err != nil {
				t.Fatalf("scope(preview) returned invalid structured content: %v", err)
			}

			ids := make([]engine.EditID, 0, len(sc.Edits))
			for _, e := range sc.Edits {
				if e.DiffURI != "" && e.Safety == config.Safe {
					ids = append(ids, e.ID)
				}
			}

			if len(ids) == 0 {
				return // No safe fixes, nothing to verify.
			}

			// Step 2: apply the safe edits by ID.
			applyArgs := tt.args
			applyArgs.Dir = dir
			applyArgs.Mode = engine.ProcessApply
			applyArgs.Apply = ids

			res, err = cs.CallTool(t.Context(), &mcp.CallToolParams{
				Name:      "scope",
				Arguments: applyArgs,
			})
			if err != nil {
				t.Fatalf("scope(apply) returned error: %v", err)
			}

			if res.IsError {
				msg := textResult(res)
				t.Fatalf("scope(apply) returned IsError: %s", msg)
			}

			compareGolden(t, dir)
		})
	}
}

func TestScopeApplyUnknownID(t *testing.T) {
	t.Parallel()

	log := testLogger(t, slog.LevelDebug)

	testdata := analysistest.TestData()
	s := server.NewMCPServer(log, true)

	srcDir := filepath.Join(testdata, "all")

	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS(srcDir)); err != nil {
		t.Fatalf("copying %s to %s failed: %v", srcDir, dir, err)
	}

	cs, cleanup := clientSession(t, s)
	defer cleanup()

	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{
		Name: "scope",
		Arguments: tools.ScopeArgs{
			Args:  engine.Args{Dir: dir},
			Mode:  engine.ProcessApply,
			Apply: []engine.EditID{123},
		},
	})
	if err != nil {
		t.Fatalf("scope(apply) returned transport error: %v", err)
	}

	if !res.IsError {
		t.Fatal("scope(apply) with unknown ID expected IsError, got none")
	}
}

func TestPatchShadow(t *testing.T) {
	t.Parallel()

	log := testLogger(t, slog.LevelDebug)

	testdata := analysistest.TestData()
	s := server.NewMCPServer(log, true)

	srcDir := filepath.Join(testdata, "rename")

	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS(srcDir)); err != nil {
		t.Fatalf("copying %s to %s failed: %v", srcDir, dir, err)
	}

	cs, cleanup := clientSession(t, s)
	defer cleanup()

	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{
		Name: guidance.ShadowToolName,
		Arguments: tools.ShadowArgs{
			Args:    engine.Args{Dir: dir},
			Renames: typeutil.RenameMap{{Name: "ShadowedErr"}: {"err": {"outerErr"}}},
			Write:   true,
		},
	})
	if err != nil {
		t.Fatalf("shadow() returned error: %v", err)
	}

	if res.IsError {
		msg := textResult(res)
		t.Fatalf("shadow() returned IsError: %s", msg)
	}

	compareGolden(t, dir)
}

func compareGolden(tb testing.TB, dir string) {
	tb.Helper()

	files, err := os.ReadDir(dir)
	if err != nil {
		tb.Fatalf("reading directory %s failed: %v", dir, err)
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".go" {
			continue
		}

		fixed := filepath.Join(dir, file.Name())
		golden := fixed + ".golden"

		wantF, err := os.ReadFile(golden)
		if err != nil {
			tb.Fatalf("golden file %s not found: %v", golden, err)
		}

		gotF, err := os.ReadFile(fixed)
		if err != nil {
			tb.Fatalf("fixed file %s not found: %v", fixed, err)
		}

		if got, want := string(gotF), string(wantF); got != want {
			udiff := diff.Unified(fixed, golden, got, want)
			tb.Errorf("fixed %s does not match golden file %s.\n--- got ---\n%s\n--- want ---\n%s\n--- diff ---\n%s",
				fixed, golden, got, want, udiff)
		}
	}
}
