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
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/guidance"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/help"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/mcputil"
)

func addHelpTool(server *mcp.Server) {
	tool := &mcp.Tool{
		Name:        guidance.HelpToolName,
		Title:       "Help",
		Description: "Retrieve ScopeGuard help and reference material.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}

	overrides := CommonSchemaOverrides

	c := HelpContext{}
	handler := mcputil.WrapStringTool(tool, c.Help, overrides)

	server.AddTool(tool, handler)
}

// HelpContext represents a context for retrieving help and reference material in the MCP framework.
type HelpContext struct{}

// HelpTopic is a help topic.
type HelpTopic string

// HelpArgs are the input arguments of the [Help] tool.
type HelpArgs struct {
	Topic HelpTopic `json:"topic,omitempty" jsonschema:"Help topic to retrieve. Omit to retrieve all topics concatenated."`
}

// Help is an MCP tool that returns ScopeGuard reference material for a single
// topic. The topic is validated upstream by the schema enum.
func (HelpContext) Help(_ context.Context, _ *mcp.CallToolRequest, args HelpArgs) (string, []mcp.Content, error) {
	text, err := help.ReadTopic(string(args.Topic))
	if err != nil {
		return "", nil, err
	}

	return text, nil, nil
}

// HelpPrefix is the URI prefix used to identify help topic resources.
const HelpPrefix = "help:///"

// helpTopic implements [mcp.ResourceHandler].
func helpTopic(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	if !strings.HasPrefix(req.Params.URI, HelpPrefix) {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	topic := req.Params.URI[len(HelpPrefix):]

	text, err := help.ReadTopic(topic)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "text/markdown",
			Text:     text,
		}},
	}, nil
}
