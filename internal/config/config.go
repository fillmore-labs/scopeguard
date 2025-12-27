// Copyright 2025-2026 Oliver Eikemeier. All Rights Reserved.
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

package config

// AnalyzerFlags represents specific analyzers.
type AnalyzerFlags uint8

const (
	// ScopeAnalyzer enables scope-based analysis for identifying variable declarations and usage.
	ScopeAnalyzer AnalyzerFlags = 1 << iota

	// ShadowAnalyzer enables analysis to detect shadowed variable declarations.
	ShadowAnalyzer

	// NestedAssignAnalyzer enables the analysis of nested assignments.
	NestedAssignAnalyzer
)

// Config represents configuration options for the analyzers.
type Config uint8

const (
	// IncludeGenerated specifies whether to include analysis of generated files.
	IncludeGenerated Config = 1 << iota

	// CombineDeclarations determines whether to combine declarations when moving to init statements.
	CombineDeclarations

	// Conservative indicates that moves should be conservative.
	Conservative

	// RenameVariables indicates that shadowed variables should be renamed.
	RenameVariables
)
