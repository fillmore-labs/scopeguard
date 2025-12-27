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

package gclplugin_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	scopeguard "fillmore-labs.com/scopeguard/analyzer"
	. "fillmore-labs.com/scopeguard/gclplugin"
)

const allSettings = `{
	"scope": true,
	"shadow": true,
	"nested-assign": true,
	"conservative": false,
	"combine": true,
	"rename": true,
	"max-lines": 10
}`

func TestSettings(t *testing.T) {
	t.Parallel()

	testCases := [...]struct {
		name     string
		settings string
		want     int
	}{
		{"all", allSettings, reflect.TypeFor[Settings]().NumField()},
		{"none", `{}`, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dec := json.NewDecoder(strings.NewReader(tc.settings))
			dec.DisallowUnknownFields()

			var s Settings
			if err := dec.Decode(&s); err != nil {
				t.Fatalf("Can't decode settings: %v", err)
			}

			if got := s.Options(); len(got) != tc.want {
				t.Errorf("Got %d options: %s, want %d", len(got), scopeguard.Options(got).LogValue(), tc.want)
			}
		})
	}
}
