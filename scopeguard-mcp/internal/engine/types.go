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

package engine

import (
	"strings"

	"fillmore-labs.com/scopeguard/internal/typeutil"
)

// Args are input arguments.
type Args struct {
	Dir       string                   `json:"dir"                 jsonschema:"Absolute path to the directory to analyze. By default, only the package directly in this folder is analyzed. To analyze all subdirectories recursively, you MUST also set packages: ['./...']."`
	Functions []typeutil.LocalFuncName `json:"functions,omitempty" jsonschema:"Optional: restrict analysis to specific functions. For functions use the bare name, prefix a method with the receiver type (e.g. ['ProcessData', 'Server.HandleRequest']). Omit for multi-package analysis."`
	Packages  []string                 `json:"packages,omitempty"  jsonschema:"Optional package patterns. Omit to analyze only the package in dir. Provide ['./...'] to perform a recursive, multi-package analysis across the entire directory tree."`
}

// Position represents the location of an issue, including its package, function, file, and line number.
type Position struct {
	Package  string `json:"package"  jsonschema:"Import path of the package containing the issue"`
	File     string `json:"file"     jsonschema:"File path where the issue was found"`
	Function string `json:"function" jsonschema:"Name of the function containing the issue"`
	Line     int    `json:"line"     jsonschema:"Line number of the issue"`
}

// Compare compares two Position instances lexicographically by file path and numerically by line number.
func (a Position) Compare(b Position) int {
	if c := strings.Compare(a.File, b.File); c != 0 {
		return c
	}

	return a.Line - b.Line
}
