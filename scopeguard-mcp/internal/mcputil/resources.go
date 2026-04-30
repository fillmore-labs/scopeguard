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
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
)

// ResultBuilder assists in building an MCP CallToolResult that can optionally include embedded resources.
type ResultBuilder struct {
	diffstore map[string]string
	state     *ServerState
	content   []mcp.Content
}

// NewResultBuilder creates a new [ResultBuilder]. If  [ServerState.EmbeddedResources]
// is enabled, it embeds diffs as resource, pre-allocating 'capacity'.
func (s *ServerState) NewResultBuilder(capacity int) ResultBuilder {
	if s.InlineResources {
		return ResultBuilder{}
	}

	return ResultBuilder{
		content:   make([]mcp.Content, 0, capacity),
		diffstore: make(map[string]string, capacity),
		state:     s,
	}
}

// EmbedDiff registers a diff as an embedded resource and returns the modified diff and URI.
// If embedded resources are disabled or the diff is empty, it returns the original diff and an empty URI.
func (b *ResultBuilder) EmbedDiff(diff string, id engine.EditID) (string, string) {
	if diff == "" || b.diffstore == nil {
		return diff, ""
	}

	// Resources do not need to map to an actual physical filesystem.
	//
	// https://modelcontextprotocol.io/specification/2025-11-25/server/resources#file//
	diffURI := fmt.Sprintf("file:///%s.diff", id)

	b.diffstore[diffURI] = diff

	b.content = append(b.content, &mcp.EmbeddedResource{
		Resource: diff2Resource(diff, diffURI),
		Annotations: &mcp.Annotations{
			Audience: []mcp.Role{"assistant"},
			Priority: 0.9,
		},
	})

	return "", diffURI
}

// diff2Resource creates an mcp.ResourceContents object from a diff string and URI.
func diff2Resource(diff, diffURI string) *mcp.ResourceContents {
	return &mcp.ResourceContents{
		URI:      diffURI,
		MIMEType: "text/x-diff",
		Text:     diff,
	}
}

// ExtraContent finalizes the response by committing the local diffstore to the server state and
// serializing the tool output into the primary text content block.
func (b *ResultBuilder) ExtraContent(session *mcp.ServerSession) []mcp.Content {
	if len(b.diffstore) == 0 {
		return nil
	}

	b.state.commitStore(session, b.diffstore)

	return b.content
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
