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

package analyzer

import (
	"flag"

	"fillmore-labs.com/scopeguard/internal/config"
)

// RegisterFlags binds the [Options] values to command line flag values.
// A nil flag set value defaults to the program's command line.
func registerFlags(flags *flag.FlagSet, r *runOptions) {
	if flags == nil {
		flags = flag.CommandLine
	}

	analyzers := analyzeFlags[config.AnalyzerFlags]{
		{config.ScopeAnalyzer, "scope", "scope analysis"},
		{config.ShadowAnalyzer, "shadow", "shadow analysis"},
		{config.NestedAssignAnalyzer, "nested-assign", "nested assign analysis"},
	}

	config := analyzeFlags[config.Config]{
		{config.IncludeGenerated, "generated", "check generated files"},
		{config.Conservative, "conservative", "enable conservative scope analysis"},
		{config.CombineDeclarations, "combine", "combine declaration when moving to initializers"},
		{config.RenameVariables, "rename", "rename shadowed variables (experimental)"},
	}

	analyzers.register(flags, &r.analyzers)
	config.register(flags, &r.behavior)
	flags.IntVar(&r.maxLines, "max-lines", r.maxLines, "maximum declaration lines for moving to initializers")
}

type analyzeFlags[T ~uint8] []struct {
	flag        T
	name, usage string
}

func (a analyzeFlags[T]) register(flags *flag.FlagSet, b *config.BitMask[T]) {
	for _, f := range a {
		flags.Var(boolValue[T, *config.BitMask[T]]{b, f.flag}, f.name, f.usage)
	}
}
