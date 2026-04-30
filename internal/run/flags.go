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

package run

import (
	"flag"
	"strconv"

	"fillmore-labs.com/scopeguard/internal/config"
)

// RegisterFlags binds the [Options] values to command line flag values.
// A nil flag set value defaults to the program's command line.
func (o *Options) registerFlags(fs *flag.FlagSet) {
	fs.Var(o.Analyzers.BoolFlag(config.ScopeAnalyzer), "scope", "scope analysis")
	fs.Var(o.Analyzers.BoolFlag(config.ShadowAnalyzer), "shadow", "shadow analysis")
	fs.Var(o.Analyzers.BoolFlag(config.NestedAssignAnalyzer), "nested-assign", "nested assign analysis")

	fs.Var(o.Behaviors.BoolFlag(config.IncludeGenerated), "generated", "check generated files")
	fs.Var(o.Behaviors.BoolFlag(config.CombineDeclarations), "combine", "combine declaration when moving to initializers")
	fs.Var(o.Behaviors.BoolFlag(config.RenameVariables), "rename", "rename shadowed variables")

	// diagnostics
	fs.Var((&o.Filters[0]).BoolFlag(config.Unsafe|config.Breaking), "unsafe-diagnostics", "add diagnostics for moves that have no automatic fix (i.e. unsafe moves)")
	// fix
	fs.Var((&o.Filters[1]).BoolFlag(config.Unsafe), "unsafe", "also apply fixes for moves that may change evaluation order relative to side effects")

	fs.IntVar(&o.MaxLines, "max-lines", o.MaxLines, "maximum declaration lines for moving to initializers")

	fs.Func("conservative", "deprecated: use -unsafe-diagnostics instead", func(s string) error {
		v, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}

		o.Filters[0].Set(config.Unsafe|config.Breaking, !v)

		return nil
	})
}
