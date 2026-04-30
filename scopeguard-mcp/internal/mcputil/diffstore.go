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

package mcputil

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// AddDiffStore registers a resource template for serving the provided [ServerState].
func AddDiffStore(server *mcp.Server, s *ServerState) {
	template := &mcp.ResourceTemplate{
		Name:        "diff_store",
		Title:       "Diff Store",
		Description: "Previews of fixes from the last call",
		MIMEType:    "text/x-diff",
		URITemplate: "file:///{ID}.diff",
	}

	server.AddResourceTemplate(template, s.DiffStore)
}

// DiffStore provides the embedded resources on request.
//
// [MCP Spec]: “Servers that use embedded resources SHOULD implement the resources capability”
//
// [MCP Spec]: https://modelcontextprotocol.io/specification/2025-11-25/server/tools#embedded-resources
func (s *ServerState) DiffStore(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	store, ok := s.getStore(req.Session)
	if !ok {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	diff, ok := store.diffs[req.Params.URI]
	if !ok {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{diff2Resource(diff, req.Params.URI)},
	}, nil
}
