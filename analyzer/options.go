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
)

// Option configures specific behavior of a [New] scopeguard analyzer.
type Option interface {
	apply(r *runOptions)
	LogAttr() slog.Attr
}

// Options is a list of [Option] values that itself satisfies the [Option] interface.
type Options []Option

// LogValue implements [slog.LogValuer].
func (o Options) LogValue() slog.Value {
	as := make([]slog.Attr, 0, len(o))
	as = appendOptions(as, o)

	return slog.GroupValue(as...)
}

func appendOptions(as []slog.Attr, o Options) []slog.Attr {
	for _, opt := range o {
		switch opt := opt.(type) {
		case nil:
			as = append(as, slog.String("nil", "<nil>"))

		case Options:
			as = appendOptions(as, opt)

		default:
			as = append(as, opt.LogAttr())
		}
	}

	return as
}

func (o Options) apply(r *runOptions) {
	for _, opt := range o {
		if opt == nil {
			continue
		}

		opt.apply(r)
	}
}

// LogAttr is for logging with [slog.Logger.LogAttrs].
func (o Options) LogAttr() slog.Attr {
	return slog.Any("options", o)
}

// WithGenerated is an [Option] to configure diagnostics in generated files.
func WithGenerated(generated bool) Option { return generatedOption{generated: generated} }

type generatedOption struct{ generated bool }

func (o generatedOption) apply(r *runOptions) {
	r.behavior.Set(config.IncludeGenerated, o.generated)
}

func (o generatedOption) LogAttr() slog.Attr {
	return slog.Bool("generated", o.generated)
}

// WithMaxLines is an [Option] to configure the maximum declaration size for moving to control flow initializers.
func WithMaxLines(maxLines int) Option { return maxLinesOption{maxLines: maxLines} }

type maxLinesOption struct{ maxLines int }

func (o maxLinesOption) apply(r *runOptions) {
	r.maxLines = o.maxLines
}

func (o maxLinesOption) LogAttr() slog.Attr {
	return slog.Int("maxLines", o.maxLines)
}

// WithScope is an [Option] to configure whether scope checks are enabled.
func WithScope(scope bool) Option {
	return scopeOption{scope: scope}
}

type scopeOption struct{ scope bool }

func (o scopeOption) apply(r *runOptions) {
	r.analyzers.Set(config.ScopeAnalyzer, o.scope)
}

func (o scopeOption) LogAttr() slog.Attr {
	return slog.Bool("scope", o.scope)
}

// WithShadow is an [Option] to configure whether shadow checks are enabled.
func WithShadow(shadow bool) Option {
	return shadowOption{shadow: shadow}
}

type shadowOption struct{ shadow bool }

func (o shadowOption) apply(r *runOptions) {
	r.analyzers.Set(config.ShadowAnalyzer, o.shadow)
}

func (o shadowOption) LogAttr() slog.Attr {
	return slog.Bool("shadow", o.shadow)
}

// WithNestedAssign is an [Option] to configure whether nested assign checks are enabled.
func WithNestedAssign(nestedAssign bool) Option {
	return nestedAssignOption{nestedAssign: nestedAssign}
}

type nestedAssignOption struct{ nestedAssign bool }

func (o nestedAssignOption) apply(r *runOptions) {
	r.analyzers.Set(config.NestedAssignAnalyzer, o.nestedAssign)
}

func (o nestedAssignOption) LogAttr() slog.Attr {
	return slog.Bool("nested-assign", o.nestedAssign)
}

// WithConservative is an [Option] to only permit moves without potential side effects.
func WithConservative(conservative bool) Option {
	return conservativeOption{conservative: conservative}
}

type conservativeOption struct{ conservative bool }

func (o conservativeOption) apply(r *runOptions) {
	r.behavior.Set(config.Conservative, o.conservative)
}

func (o conservativeOption) LogAttr() slog.Attr {
	return slog.Bool("conservative", o.conservative)
}

// WithCombine is an [Option] to configure combining declaration when moving to control flow initializers.
func WithCombine(combine bool) Option { return combineOption{combine: combine} }

type combineOption struct{ combine bool }

func (o combineOption) apply(r *runOptions) {
	r.behavior.Set(config.CombineDeclarations, o.combine)
}

func (o combineOption) LogAttr() slog.Attr {
	return slog.Bool("combine", o.combine)
}

// WithRename is an [Option] to configure renaming shadowed variables.
func WithRename(rename bool) Option { return renameOption{rename: rename} }

type renameOption struct{ rename bool }

func (o renameOption) apply(r *runOptions) {
	r.behavior.Set(config.RenameVariables, o.rename)
}

func (o renameOption) LogAttr() slog.Attr {
	return slog.Bool("rename", o.rename)
}
