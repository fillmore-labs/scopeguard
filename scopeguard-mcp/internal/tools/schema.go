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
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"

	"fillmore-labs.com/scopeguard/internal/config"
	"fillmore-labs.com/scopeguard/internal/typeutil"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/help"
)

// CommonSchemaOverrides are overrides for [jsonschema.For] schema generation.
var CommonSchemaOverrides = func() map[reflect.Type]*jsonschema.Schema {
	const descriptionNotInSchema = ""

	stringSchema := &jsonschema.Schema{Type: "string"}

	return map[reflect.Type]*jsonschema.Schema{
		reflect.TypeFor[engine.SafetyTiers](): {
			Types: []string{"null", "array"}, Items: enumSchema(engine.ValidTiers(), "Safety tiers."),
		},

		reflect.TypeFor[engine.ProcessMode](): enumSchema(engine.ProcessModes[:], descriptionNotInSchema),

		reflect.TypeFor[engine.EditID](): {
			Type: "string", Description: "Fix ID.",
		},

		reflect.TypeFor[typeutil.LocalFuncName](): {
			Type: "string", Description: "Local function name.",
		},

		reflect.TypeFor[HelpTopic](): enumSchema(help.AllTopics(), descriptionNotInSchema),

		reflect.TypeFor[config.Safety](): enumSchema(config.ValidSafeties(), descriptionNotInSchema),

		reflect.TypeFor[engine.Issue](): enumSchema(engine.ValidIssueNames(), descriptionNotInSchema),

		reflect.TypeFor[typeutil.RenameMap](): {
			Type: "object", Description: descriptionNotInSchema, AdditionalProperties: &jsonschema.Schema{
				Type: "object", Description: "outer variable name", AdditionalProperties: &jsonschema.Schema{
					Type: "array", Items: stringSchema, Description: "list of replacement names",
				},
			},
		},
	}
}()

func enumSchema(values []string, description string) *jsonschema.Schema {
	enum := make([]any, len(values))
	for i, value := range values {
		enum[i] = value
	}

	return &jsonschema.Schema{Type: "string", Enum: enum, Description: description}
}
