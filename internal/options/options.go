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

package options

import "log/slog"

// Option is a single functional option.
type Option[T any] interface {
	Apply(opts T)
	LogAttr() slog.Attr
}

// Join creates a new Option joining the provided Option values.
//
// The result implements [slog.LogValuer], so the following is evaluated lazily:
//
//	slog.LogAttrs(ctx, slog.LevelInfo, "settings", Join(opts...).LogAttr())
func Join[T any, O Option[T]](opts []O) Options[T, O] {
	return Options[T, O](opts)
}

// Options is a collection of functional Options.
type Options[T any, O Option[T]] []O

// Apply applies each Option in the list.
func (o Options[T, _]) Apply(opts T) {
	for _, opt := range o {
		if any(opt) == nil {
			continue // skip nil options
		}

		opt.Apply(opts)
	}
}

const optionsName = "options"

// LogAttr returns a lazy [slog.Attr] for logging.
func (o Options[_, _]) LogAttr() slog.Attr {
	return slog.Any(optionsName, o)
}

// LogValue implements [slog.LogValuer].
func (o Options[T, O]) LogValue() slog.Value {
	as := make([]slog.Attr, 0, len(o)) // Optimize for non-flattening case
	for _, opt := range o {
		if any(opt) == nil {
			continue // skip nil options
		}

		la := opt.LogAttr()
		if la.Key == optionsName {
			if v := la.Value.Resolve(); v.Kind() == slog.KindGroup {
				// aggregate group values
				as = append(as, v.Group()...)
				continue
			}
		}

		as = append(as, la)
	}

	return slog.GroupValue(as...)
}
