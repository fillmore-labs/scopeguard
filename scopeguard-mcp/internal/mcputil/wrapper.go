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
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolHandler is a generic function type for handling MCP tool requests.
type ToolHandler[In, Out any] func(ctx context.Context, request *mcp.CallToolRequest, input In) (Out, []mcp.Content, error)

type wrapperContext[In, Out any] struct {
	inputResolved  *jsonschema.Resolved
	outputResolved *jsonschema.Resolved
	handler        ToolHandler[In, Out]
}

// WrapTool wraps the provided tool handler into a [mcp.ToolHandler].
func WrapTool[In, Out any](tool *mcp.Tool, handler ToolHandler[In, Out], overrides map[reflect.Type]*jsonschema.Schema) mcp.ToolHandler {
	c := wrapperContext[In, Out]{
		handler: handler,
	}

	var err error

	in := reflect.TypeFor[In]()

	tool.InputSchema, c.inputResolved, err = resolvedSchema(in, overrides)
	if err != nil {
		panic(fmt.Sprintf("input schema of %s: %v", in, err))
	}

	out := reflect.TypeFor[Out]()

	tool.OutputSchema, c.outputResolved, err = resolvedSchema(out, overrides)
	if err != nil {
		panic(fmt.Sprintf("output schema of %s: %v", out, err))
	}

	return c.Tool
}

func (c wrapperContext[In, Out]) Tool(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var in In
	if err := unmarshalParams(req, &in, c.inputResolved); err != nil {
		return returnErr(err)
	}

	out, extra, err := c.handler(ctx, req, in)
	if err != nil {
		return returnErr(err)
	}

	return marshalResult(out, extra, c.outputResolved)
}

type wrapperStringContext[In any] struct {
	inputResolved *jsonschema.Resolved
	handler       ToolHandler[In, string]
}

// WrapStringTool wraps the given tool handler with string output into a [mcp.ToolHandler].
func WrapStringTool[In any](tool *mcp.Tool, handler ToolHandler[In, string], overrides map[reflect.Type]*jsonschema.Schema) mcp.ToolHandler {
	c := wrapperStringContext[In]{
		handler: handler,
	}

	var err error
	in := reflect.TypeFor[In]()

	tool.InputSchema, c.inputResolved, err = resolvedSchema(in, overrides)
	if err != nil {
		panic(fmt.Sprintf("input schema of %s: %v", in, err))
	}

	return c.Tool
}

func (c wrapperStringContext[In]) Tool(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var in In
	if err := unmarshalParams(req, &in, c.inputResolved); err != nil {
		return returnErr(err)
	}

	text, extra, err := c.handler(ctx, req, in)
	if err != nil {
		return returnErr(err)
	}

	content := make([]mcp.Content, 1, len(extra)+1)
	content[0] = &mcp.TextContent{Text: text}
	content = append(content, extra...)

	return &mcp.CallToolResult{
		Content: content,
	}, nil
}

func unmarshalParams(req *mcp.CallToolRequest, in any, resolved *jsonschema.Resolved) error {
	var input json.RawMessage
	if req.Params.Arguments != nil {
		input = req.Params.Arguments
	}

	// Validate input and apply defaults.
	var err error

	input, err = applySchema(input, resolved)
	if err != nil {
		return fmt.Errorf("validating \"arguments\": %v", err)
	}

	if input == nil {
		return nil
	}

	return json.Unmarshal(input, in)
}

func marshalResult(out any, extracontent []mcp.Content, resolved *jsonschema.Resolved) (*mcp.CallToolResult, error) {
	outJSON, err := applySchema(out, resolved)
	if err != nil {
		return returnErr(err)
	}

	// Return the serialized JSON in a TextContent block, as the spec suggests:
	// https://modelcontextprotocol.io/specification/2025-11-25/server/tools#structured-content.
	content := make([]mcp.Content, 1, len(extracontent)+1)
	content[0] = &mcp.TextContent{Text: string(outJSON)}
	content = append(content, extracontent...)

	return &mcp.CallToolResult{
		Content:           content,
		StructuredContent: json.RawMessage(outJSON),
	}, nil
}

func resolvedSchema(typ reflect.Type, schemaOverrides map[reflect.Type]*jsonschema.Schema) (any, *jsonschema.Resolved, error) {
	schema, err := schemaOf(typ, schemaOverrides)
	if err != nil {
		return nil, nil, err
	}

	if schema == nil {
		return nil, nil, nil
	}

	resolved, err := schema.Resolve(&jsonschema.ResolveOptions{ValidateDefaults: true})
	if err != nil {
		return nil, nil, err
	}

	return schema, resolved, nil
}

func schemaOf(typ reflect.Type, overrides map[reflect.Type]*jsonschema.Schema) (*jsonschema.Schema, error) {
	if typ == reflect.TypeFor[any]() {
		return nil, nil
	}

	// Make it non-optional
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	js, err := jsonschema.ForType(typ, &jsonschema.ForOptions{TypeSchemas: overrides})
	if err != nil {
		return nil, fmt.Errorf("schema of %T: %w", typ, err)
	}

	if js.Type != "object" {
		return nil, fmt.Errorf("schema of %T must be 'object', is %s", typ, js.Type)
	}

	return js, nil
}

func applySchema(data any, resolved *jsonschema.Resolved) ([]byte, error) {
	outJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshaling output: %w", err)
	}

	if resolved == nil {
		return outJSON, nil
	}

	var outval map[string]any
	if err := json.Unmarshal(outJSON, &outval); err != nil {
		return nil, fmt.Errorf("unmarshaling output: %w", err)
	}

	if err := resolved.ApplyDefaults(&outval); err != nil {
		return nil, fmt.Errorf("applying defaults: %w", err)
	}

	if err := resolved.Validate(outval); err != nil {
		return nil, fmt.Errorf("validating tool output: %w", err)
	}

	outJSON, err = json.Marshal(outval)
	if err != nil {
		return nil, fmt.Errorf("marshaling with defaults: %w", err)
	}

	return outJSON, nil
}

func returnErr(err error) (*mcp.CallToolResult, error) {
	var res mcp.CallToolResult
	res.SetError(err)

	return &res, nil
}
