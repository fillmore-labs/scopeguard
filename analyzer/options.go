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

	"fillmore-labs.com/scopeguard/internal/analyze"
)

// makeOptions returns a [analyze.Options] struct with overriding [Options] applied.
func makeOptions(opts Options) *analyze.Options {
	o := analyze.DefaultOptions()
	opts.apply(o)

	return o
}

// Option configures specific behavior of a [New] scopeguard [analysis.Analyzer].
type Option interface {
	apply(opts *analyze.Options)
	LogAttr() slog.Attr
}

// Options is a list of [Option] values that also satisfies the [Option] interface.
type Options []Option

func (o Options) apply(opts *analyze.Options) {
	for _, opt := range o {
		opt.apply(opts)
	}
}

// LogValue implements [slog.LogValuer].
func (o Options) LogValue() slog.Value {
	as := make([]slog.Attr, 0, len(o))
	for _, opt := range o {
		as = append(as, opt.LogAttr())
	}

	return slog.GroupValue(as...)
}

// LogAttr returns a [slog.Attr] for logging.
func (o Options) LogAttr() slog.Attr {
	return slog.Any("options", o)
}

// WithGenerated is an [Option] to configure diagnostics in generated files.
func WithGenerated(generated bool) Option { return generatedOption{generated: generated} }

type generatedOption struct{ generated bool }

func (o generatedOption) apply(opts *analyze.Options) {
	opts.Generated = o.generated
}

func (o generatedOption) LogAttr() slog.Attr {
	return slog.Bool("generated", o.generated)
}

// WithMaxLines is an [Option] to configure diagnostics in generated files.
func WithMaxLines(maxLines int) Option { return maxLinesOption{maxLines: maxLines} }

type maxLinesOption struct{ maxLines int }

func (o maxLinesOption) apply(opts *analyze.Options) {
	opts.MaxLines = o.maxLines
}

func (o maxLinesOption) LogAttr() slog.Attr {
	return slog.Int("maxLines", o.maxLines)
}
