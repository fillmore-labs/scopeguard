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

package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/help"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/mcputil"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/tools"
)

// NewMCPServer initializes and returns a new instance of a scopeguard MCP server.
func NewMCPServer(log *slog.Logger, inlineResources bool) *mcp.Server {
	version := "unknown"
	if info, ok := debug.ReadBuildInfo(); ok {
		version = info.Main.Version
	}

	impl := &mcp.Implementation{
		Name:        "scopeguard",
		Title:       "ScopeGuard MCP",
		Description: "The MCP server for the ScopeGuard analyzer",
		Version:     version,
		WebsiteURL:  "https://fillmore-labs.com/scopeguard/scopeguard-mcp",
	}

	opts := &mcp.ServerOptions{
		Instructions: help.Instructions,
		Capabilities: &mcp.ServerCapabilities{
			Resources: &mcp.ResourceCapabilities{},
			Tools:     &mcp.ToolCapabilities{},
		},
		Logger: log,
	}

	server := mcp.NewServer(impl, opts)

	state := &mcputil.ServerState{InlineResources: inlineResources}
	AddTools(server, state)

	return server
}

// AddTools registers multiple tools, including Analyze, Scope, Shadow, and Help.
func AddTools(server *mcp.Server, state *mcputil.ServerState) {
	mcputil.AddDiffStore(server, state)

	tools.AddAnalyzeTool(server)
	tools.AddScopeTool(server, state)
	tools.AddShadowTool(server, state)
	tools.AddHelpTool(server)
	tools.AddHelpTopics(server)
}

// Options for the scopeguard MCP server.
type Options struct {
	HTTPAddr        string
	LogDir          string
	InlineResources bool
}

// DefaultOptions creates default options for the scopeguard MCP server.
func DefaultOptions() *Options {
	return &Options{}
}

// Run runs the MCP server with the given options.
func (s *Options) Run(ctx context.Context) error {
	var logger *slog.Logger

	if s.LogDir != "" {
		if err := os.MkdirAll(s.LogDir, 0o750); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		timestamp := time.Now().Format("20060102-150405")
		pid := os.Getpid()
		filename := fmt.Sprintf("scopeguard-%s-%d.log", timestamp, pid)
		fullPath := filepath.Join(s.LogDir, filename)

		logFile, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
		if err != nil {
			return fmt.Errorf("failed to create log file: %w", err)
		}
		defer logFile.Close()

		logger = slog.New(slog.NewTextHandler(logFile, nil))
	}

	server := NewMCPServer(logger, s.InlineResources)

	if s.HTTPAddr == "" {
		// Serve over stdio.
		transport := &mcp.StdioTransport{}
		return server.Run(ctx, transport)
	}

	ln, err := net.Listen("tcp", s.HTTPAddr)
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "ScopeGuard MCP server listening", "address", ln.Addr())

	// Serve over streamable HTTP.
	getServer := func(*http.Request) *mcp.Server { return server }
	// cors := http.NewCrossOriginProtection()
	// cors.AddInsecureBypassPattern("http://localhost")
	handler := mcp.NewStreamableHTTPHandler(getServer, &mcp.StreamableHTTPOptions{
		// Stateless: true,
		Logger: logger,
	})

	httpServer := &http.Server{
		Addr:              s.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	shutdownDone := make(chan struct{})
	defer close(shutdownDone)

	// Shut down gracefully when the caller's context is canceled.
	go shutdownWhenCancelled(ctx, httpServer, shutdownDone) // #nosec:G118

	if err := httpServer.Serve(ln); err != http.ErrServerClosed {
		return err
	}

	return nil
}

// httpShutdownTimeout bounds how long Run waits for in-flight HTTP requests
// to complete when the context is canceled.
const httpShutdownTimeout = 5 * time.Second

func shutdownWhenCancelled(ctx context.Context, s *http.Server, shutdownDone <-chan struct{}) {
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), httpShutdownTimeout)
		defer cancel()

		_ = s.Shutdown(shutdownCtx)

	case <-shutdownDone:
	}
}
