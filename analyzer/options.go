// Copyright 2025 Oliver Eikemeier. All Rights Reserved.
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

	"fillmore-labs.com/scopeguard/analyzer/level"
	"fillmore-labs.com/scopeguard/internal/analyze"
)

// makeOptions returns a [analyze.Options] struct with overriding [Options] applied.
func makeOptions(opts Options) *analyze.Options {
	o := analyze.DefaultOptions()
	opts.apply(o)

	return o
}

// Option configures specific behavior of a [New] scopeguard analyzer.
type Option interface {
	apply(opts *analyze.Options)
	logAttr() slog.Attr
}

// Options is a list of [Option] values that itself satisfies the [Option] interface.
type Options []Option

// LogValue implements [slog.LogValuer].
func (o Options) LogValue() slog.Value {
	as := make([]slog.Attr, 0, len(o))
	for _, opt := range o {
		as = append(as, opt.logAttr())
	}

	return slog.GroupValue(as...)
}

func (o Options) apply(opts *analyze.Options) {
	for _, opt := range o {
		opt.apply(opts)
	}
}

func (o Options) logAttr() slog.Attr {
	return slog.Any("options", o)
}

// WithGenerated is an [Option] to configure diagnostics in generated files.
func WithGenerated(generated bool) Option { return generatedOption{generated: generated} }

type generatedOption struct{ generated bool }

func (o generatedOption) apply(opts *analyze.Options) {
	opts.Generated = o.generated
}

func (o generatedOption) logAttr() slog.Attr {
	return slog.Bool("generated", o.generated)
}

// WithMaxLines is an [Option] to configure the maximum declaration size for moving to control flow initializers.
func WithMaxLines(maxLines int) Option { return maxLinesOption{maxLines: maxLines} }

type maxLinesOption struct{ maxLines int }

func (o maxLinesOption) apply(opts *analyze.Options) {
	opts.MaxLines = o.maxLines
}

func (o maxLinesOption) logAttr() slog.Attr {
	return slog.Int("maxLines", o.maxLines)
}

// WithScope is an [Option] to configure which scope checks are enabled.
func WithScope(scope level.Scope) Option {
	return scopeOption{scope: scope}
}

type scopeOption struct{ scope level.Scope }

func (o scopeOption) apply(opts *analyze.Options) {
	opts.ScopeLevel = o.scope
}

func (o scopeOption) logAttr() slog.Attr {
	text, err := o.scope.MarshalText()
	if err != nil {
		return slog.String("scope", err.Error())
	}

	return slog.String("scope", string(text))
}

// WithShadow is an [Option] to configure which shadow checks are enabled.
func WithShadow(shadow level.Shadow) Option {
	return shadowOption{shadow: shadow}
}

type shadowOption struct{ shadow level.Shadow }

func (o shadowOption) apply(opts *analyze.Options) {
	opts.ShadowLevel = o.shadow
}

func (o shadowOption) logAttr() slog.Attr {
	text, err := o.shadow.MarshalText()
	if err != nil {
		return slog.String("shadow", err.Error())
	}

	return slog.String("shadow", string(text))
}

// WithNestedAssign is an [Option] to configure which nested assign checks are enabled.
func WithNestedAssign(nestedAssign level.NestedAssign) Option {
	return nestedAssignOption{nestedAssign: nestedAssign}
}

type nestedAssignOption struct{ nestedAssign level.NestedAssign }

func (o nestedAssignOption) apply(opts *analyze.Options) {
	opts.NestedAssign = o.nestedAssign
}

func (o nestedAssignOption) logAttr() slog.Attr {
	text, err := o.nestedAssign.MarshalText()
	if err != nil {
		return slog.String("nested-assign", err.Error())
	}

	return slog.String("nested-assign", string(text))
}
