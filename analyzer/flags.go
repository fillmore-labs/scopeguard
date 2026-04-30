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
	"fillmore-labs.com/scopeguard/internal/run"
)

// RegisterFlags binds the [Options] values to command line flag values.
// A nil flag set value defaults to the program's command line.
func registerFlags(flags *flag.FlagSet, r *run.Options) {
	if flags == nil {
		flags = flag.CommandLine
	}

	analyzers := analyzeFlags[config.Analyzers]{
		{config.ScopeAnalyzer, "scope", "scope analysis"},
		{config.ShadowAnalyzer, "shadow", "shadow analysis"},
		{config.NestedAssignAnalyzer, "nested-assign", "nested assign analysis"},
	}
	analyzers.register(flags, &r.Analyzers)

	behavior := analyzeFlags[config.Behaviors]{
		{config.IncludeGenerated, "generated", "check generated files"},
		{config.CombineDeclarations, "combine", "combine declaration when moving to initializers"},
		{config.RenameVariables, "rename", "rename shadowed variables"},
	}
	behavior.register(flags, &r.Behaviors)

	diagnosticFilter := analyzeFlags[config.SafetyFilter]{
		{config.FilterUnsafe | config.FilterBreaking, "unsafe-diagnostics", "add diagnostics for moves that have no automatic fix (i.e. unsafe moves)"},
	}
	diagnosticFilter.register(flags, &r.Filters[0])

	fixFilter := analyzeFlags[config.SafetyFilter]{
		{config.FilterUnsafe, "unsafe", "also apply fixes for moves that may change evaluation order relative to side effects"},
	}
	fixFilter.register(flags, &r.Filters[1])

	flags.IntVar(&r.MaxLines, "max-lines", r.MaxLines, "maximum declaration lines for moving to initializers")

	flags.Func("conservative", "deprecated: use -unsafe-diagnostics instead", func(s string) error {
		v, err := parseBool(s)
		if err != nil {
			return err
		}

		r.Filters[0].Set(config.FilterUnsafe|config.FilterBreaking, !v)

		return nil
	})
}

type analyzeFlags[T any] []struct {
	flag        T
	name, usage string
}

func (a analyzeFlags[T]) register(flags *flag.FlagSet, b settable[T]) {
	for _, f := range a {
		flags.Var(newFlagValue(b, f.flag), f.name, f.usage)
	}
}
