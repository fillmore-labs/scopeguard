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

import scopeguard "fillmore-labs.com/scopeguard/analyzer"

// Settings represent the configuration options for an instance of the [Plugin].
type Settings struct {
	MaxLines *int `json:"max-lines,omitzero"`
}

// options converts [Settings] into [scopeguard.Options] for the scopeguard analyzer.
// It processes boolean settings and applies them only when explicitly set (non-nil).
func options(settings Settings) scopeguard.Options {
	optionConfigs := [...]struct {
		f func(int) scopeguard.Option
		v *int
	}{
		{scopeguard.WithMaxLines, settings.MaxLines},
	}

	var options scopeguard.Options

	for _, opt := range optionConfigs {
		if opt.v != nil {
			options = append(options, opt.f(*opt.v))
		}
	}

	return options
}
