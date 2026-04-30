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
	"log/slog"

	"fillmore-labs.com/scopeguard/internal/config"
	"fillmore-labs.com/scopeguard/internal/options"
	"fillmore-labs.com/scopeguard/internal/run"
	"fillmore-labs.com/scopeguard/internal/typeutil"
)

// Option configures specific behavior of a [New] scopeguard analyzer.
type Option interface {
	Apply(opts *run.Options) error
	LogAttr() slog.Attr
}

// Join creates a new Option joining the provided Option values.
//
// The result implements [slog.LogValuer], so the following is evaluated lazily:
//
//	slog.LogAttrs(ctx, slog.LevelInfo, "settings", Join(opts...).LogAttr())
func Join(opts ...Option) Option {
	return options.Join(opts)
}

// WithScope is an [Option] to configure whether scope checks are enabled.
func WithScope(scope bool) Option {
	return analyzerOption{flag: config.ScopeAnalyzer, value: scope}
}

// WithShadow is an [Option] to configure whether shadow checks are enabled.
func WithShadow(shadow bool) Option {
	return analyzerOption{flag: config.ShadowAnalyzer, value: shadow}
}

// WithNestedAssign is an [Option] to configure whether nested assign checks are enabled.
func WithNestedAssign(nestedAssign bool) Option {
	return analyzerOption{flag: config.NestedAssignAnalyzer, value: nestedAssign}
}

// WithUnsafeDiagnostics is an [Option] to emit diagnostics for moves that have no automatic fix
// available (i.e., those that would require Unsafe to apply) so the user can review them.
func WithUnsafeDiagnostics(unsafeDiagnostics bool) Option {
	return filterOption{name: "unsafe-diagnostics", index: 0, filter: config.Unsafe | config.Breaking, value: unsafeDiagnostics}
}

// WithUnsafe is an [Option] to apply fixes for moves that may change evaluation
// order relative to side effects.
func WithUnsafe(unsafeFixes bool) Option {
	return filterOption{name: "unsafe", index: 1, filter: config.Unsafe, value: unsafeFixes}
}

type filterOption struct {
	name   string
	index  int
	filter config.Safety
	value  bool
}

func (o filterOption) Apply(r *run.Options) error {
	r.Filters[o.index].Set(o.filter, o.value)

	return nil
}

func (o filterOption) LogAttr() slog.Attr {
	return slog.Bool(o.name, o.value)
}

// WithCombine is an [Option] to configure combining declaration when moving to control flow initializers.
func WithCombine(combine bool) Option {
	return behaviorsOption{flag: config.CombineDeclarations, value: combine}
}

// WithRename is an [Option] to configure renaming shadowed variables.
func WithRename(rename bool) Option {
	return behaviorsOption{flag: config.RenameVariables, value: rename}
}

// WithGenerated is an [Option] to configure diagnostics in generated files.
func WithGenerated(generated bool) Option {
	return behaviorsOption{flag: config.IncludeGenerated, value: generated}
}

// withFunctions is an internal [Option] to restrict analysis to the named functions.
func withFunctions(functions ...typeutil.LocalFuncName) Option {
	return functionsOption{functions: functions}
}

// withRenames is an internal [Option] to suggest new names for shadowed variables.
func withRenames(renames typeutil.RenameMap) Option {
	return renamesOption{renames: renames}
}

// WithMaxLines is an [Option] to configure the maximum declaration size for moving to control flow initializers.
func WithMaxLines(maxLines int) Option { return maxLinesOption{maxLines: maxLines} }

type analyzerOption struct {
	flag  config.Analyzers
	value bool
}

func (o analyzerOption) Apply(r *run.Options) error {
	r.Analyzers.Set(o.flag, o.value)
	return nil
}

func (o analyzerOption) LogAttr() slog.Attr {
	return slog.Bool(o.flag.String(), o.value)
}

type behaviorsOption struct {
	flag  config.Behaviors
	value bool
}

func (o behaviorsOption) Apply(r *run.Options) error {
	r.Behaviors.Set(o.flag, o.value)
	return nil
}

func (o behaviorsOption) LogAttr() slog.Attr {
	return slog.Bool(o.flag.String(), o.value)
}

type functionsOption struct{ functions []typeutil.LocalFuncName }

func (o functionsOption) Apply(r *run.Options) error {
	r.Functions = append(r.Functions, o.functions...)
	return nil
}

func (o functionsOption) LogAttr() slog.Attr {
	return slog.Any("functions", o.functions)
}

type renamesOption struct {
	renames typeutil.RenameMap
}

func (o renamesOption) Apply(r *run.Options) error {
	for fn, rmap := range o.renames {
		for name, renames := range rmap {
			if r.Renames == nil {
				r.Renames = make(typeutil.RenameMap)
			}

			if r.Renames[fn] == nil {
				r.Renames[fn] = make(map[string][]string)
			}
			r.Renames[fn][name] = renames
		}
	}

	return nil
}

func (o renamesOption) LogAttr() slog.Attr {
	return slog.Any("renames", o.renames)
}

type maxLinesOption struct{ maxLines int }

func (o maxLinesOption) Apply(r *run.Options) error {
	r.MaxLines = o.maxLines
	return nil
}

func (o maxLinesOption) LogAttr() slog.Attr {
	return slog.Int("maxLines", o.maxLines)
}
