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
)

// Option configures specific behavior of a [New] scopeguard analyzer.
type Option interface {
	Apply(opts *run.Options)
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

// WithConservative is an [Option] to only permit moves without potential side effects.
func WithConservative(conservative bool) Option {
	return behaviorOption{flag: config.Conservative, value: conservative}
}

// WithCombine is an [Option] to configure combining declaration when moving to control flow initializers.
func WithCombine(combine bool) Option {
	return behaviorOption{flag: config.CombineDeclarations, value: combine}
}

// WithRename is an [Option] to configure renaming shadowed variables.
func WithRename(rename bool) Option {
	return behaviorOption{flag: config.RenameVariables, value: rename}
}

// WithGenerated is an [Option] to configure diagnostics in generated files.
func WithGenerated(generated bool) Option {
	return behaviorOption{flag: config.IncludeGenerated, value: generated}
}

// WithMaxLines is an [Option] to configure the maximum declaration size for moving to control flow initializers.
func WithMaxLines(maxLines int) Option { return maxLinesOption{maxLines: maxLines} }

type analyzerOption struct {
	flag  config.Analyzers
	value bool
}

func (o analyzerOption) Apply(r *run.Options) {
	r.Analyzers.Set(o.flag, o.value)
}

func (o analyzerOption) LogAttr() slog.Attr {
	return slog.Bool(o.flag.String(), o.value)
}

type behaviorOption struct {
	flag  config.Behavior
	value bool
}

func (o behaviorOption) Apply(r *run.Options) {
	r.Behavior.Set(o.flag, o.value)
}

func (o behaviorOption) LogAttr() slog.Attr {
	return slog.Bool(o.flag.String(), o.value)
}

type maxLinesOption struct{ maxLines int }

func (o maxLinesOption) Apply(r *run.Options) {
	r.MaxLines = o.maxLines
}

func (o maxLinesOption) LogAttr() slog.Attr {
	return slog.Int("maxLines", o.maxLines)
}
