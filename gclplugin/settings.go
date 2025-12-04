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

package gclplugin

import (
	"reflect"

	scopeguard "fillmore-labs.com/scopeguard/analyzer"
	"fillmore-labs.com/scopeguard/analyzer/level"
)

// Settings represent the configuration options for an instance of the [Plugin].
type Settings struct {
	Scope        *level.Scope        `json:"scope,omitzero"`
	Shadow       *level.Shadow       `json:"shadow,omitzero"`
	NestedAssign *level.NestedAssign `json:"nested-assign,omitzero"`
	MaxLines     *int                `json:"max-lines,omitzero"`
}

// Options converts [Settings] into [scopeguard.Options] for the scopeguard analyzer.
// It processes settings and applies them only when explicitly set (non-nil).
func (s Settings) Options() scopeguard.Options {
	settings := mapping{
		{scopeguard.WithScope, s.Scope},
		{scopeguard.WithShadow, s.Shadow},
		{scopeguard.WithNestedAssign, s.NestedAssign},
		{scopeguard.WithMaxLines, s.MaxLines},
	}

	return settings.options()
}

type mapping []struct {
	fun   any // func(T) scopeguard.Option
	value any // *T
}

func (m mapping) options() scopeguard.Options {
	var opts scopeguard.Options

	for _, opt := range m {
		// var v *T = opt.value
		// if v == nil {
		//	continue
		// }
		v := reflect.ValueOf(opt.value)
		if v.IsNil() {
			continue
		}

		// var f func(T) scopeguard.Option = opt.fun
		// result := f(*v)
		f := reflect.ValueOf(opt.fun)
		result := f.Call([]reflect.Value{v.Elem()})[0].Interface().(scopeguard.Option)

		opts = append(opts, result)
	}

	return opts
}
