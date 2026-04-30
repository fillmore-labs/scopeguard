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

package options_test

import (
	"log/slog"
	"strings"
	"testing"

	"fillmore-labs.com/scopeguard/internal/options"
)

func TestJoin_Apply(t *testing.T) {
	t.Parallel()

	var (
		o1     Option = countOpt(1)
		o2     Option = countOpt(2)
		nilOpt Option
	)

	tests := [...]struct {
		name     string
		opts     []Option
		expected int
	}{
		{"empty", nil, 0},
		{"single", []Option{o1}, 1},
		{"multiple", []Option{o1, o2}, 3},
		{"with nil", []Option{o1, nil, o2}, 3},
		{"nested", []Option{o1, Join(o2, o1), o2}, 6},
		{"nested with nil", []Option{o1, Join(o2, nilOpt), nil}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{}
			_ = Join(tt.opts...).Apply(cfg)

			if cfg.Count != tt.expected {
				t.Errorf("expected count %d, got %d", tt.expected, cfg.Count)
			}
		})
	}
}

func TestJoin_LogValuer(t *testing.T) {
	t.Parallel()

	joined := Join(countOpt(1), countOpt(2))

	var panicvalue any

	var attributes []slog.Attr

	func() {
		defer func() {
			panicvalue = recover()
		}()

		attributes = joined.LogAttr().Value.LogValuer().LogValue().Group()
	}()

	if panicvalue != nil {
		t.Fatalf("expected joined options the implement LogValuer, but %T does not", joined)
	}

	if len(attributes) != 2 {
		t.Errorf("expected 2 attributes, got %d", len(attributes))
	}
}

func TestJoin_LogValue(t *testing.T) {
	t.Parallel()

	var (
		o1     = countOpt(1)
		o2     = countOpt(2)
		nilOpt = Option(nil)
	)

	tests := [...]struct {
		name     string
		expected string
		opts     []Option
	}{
		{"empty", "msg=test\n", nil},
		{"single", "msg=test options.count=1\n", []Option{o1}},
		{"multiple", "msg=test options.count=1 options.count=2\n", []Option{o1, o2}},
		{"with nil", "msg=test options.count=1 options.count=2\n", []Option{o1, nil, o2}},
		{"nested", "msg=test options.count=1 options.count=2 options.count=1 options.count=2\n", []Option{o1, Join(o2, o1), o2}},
		{"nested with nil", "msg=test options.count=1 options.count=2\n", []Option{o1, Join(o2, nilOpt), nil}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var sb strings.Builder
			logger := slog.New(slog.NewTextHandler(&sb, nil))

			joined := Join(tt.opts...)
			logger.LogAttrs(t.Context(), slog.LevelInfo, "test", joined.LogAttr())

			if got := sb.String(); !strings.HasSuffix(got, tt.expected) {
				t.Errorf("Expected %s to end with %s", got, tt.expected)
			}
		})
	}
}

// Run with `go test -benchmem -bench=BenchmarkPlainOpts ./internal/options`.
func BenchmarkPlainOpts(b *testing.B) {
	config := new(Config)

	for opts := []countOpt{1, 2}; b.Loop(); {
		joined := options.Join(opts)
		_ = joined.Apply(config)
	}
}

type Config struct {
	Count int
}

type Option interface {
	Apply(opts *Config) error
	LogAttr() slog.Attr
	internal()
}

func Join(opts ...Option) Option { return wrapped{options.Join(opts)} }

type wrapped struct{ options.Option[*Config] }

func (wrapped) internal() {}

type countOpt int

func (c countOpt) Apply(opts *Config) error { opts.Count += int(c); return nil }

func (c countOpt) LogAttr() slog.Attr { return slog.Int("count", int(c)) }

func (c countOpt) internal() {}
