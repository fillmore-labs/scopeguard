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

package main

import "testing"

// alphaSpec returns a single-spec slice with one bit, one multi-bit alias, and
// the zero alias, used to exercise applyTrueValues.
func alphaSpec() []Spec {
	return []Spec{{
		TypeName: "Alpha",
		Bits:     []constInfo{{Const: "AlphaOne", Name: "one", value: 1}},
		Aliases: []constInfo{
			{Const: "AlphaAll", Name: "all", value: 0b11},
			{Const: "AlphaNone", Name: "zero", value: 0},
		},
	}}
}

func TestApplyTrueValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		want    string
		entries []string
		wantErr bool
	}{
		{name: "bit", entries: []string{"AlphaOne"}, want: "AlphaOne"},
		{name: "alias", entries: []string{"AlphaAll"}, want: "AlphaAll"},
		{name: "qualified", entries: []string{"Alpha.AlphaOne"}, want: "AlphaOne"},
		{name: "zero value rejected", entries: []string{"AlphaNone"}, wantErr: true},
		{name: "unknown constant", entries: []string{"Missing"}, wantErr: true},
		{name: "duplicate entries", entries: []string{"AlphaOne", "AlphaAll"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			specs := alphaSpec()

			if err := applyTrueValues(specs, tt.entries); (err != nil) != tt.wantErr {
				t.Fatalf("applyTrueValues(%v) err = %v, wantErr = %v", tt.entries, err, tt.wantErr)
			}

			if !tt.wantErr && specs[0].TrueValue != tt.want {
				t.Errorf("applyTrueValues(%v) TrueValue = %q, want %q", tt.entries, specs[0].TrueValue, tt.want)
			}
		})
	}
}

// TestApplyTrueValues_MultipleTypes covers the qualification rules that only
// apply when more than one spec is present.
func TestApplyTrueValues_MultipleTypes(t *testing.T) {
	t.Parallel()

	newSpecs := func() []Spec {
		return []Spec{
			{TypeName: "Alpha", Bits: []constInfo{{Const: "AlphaOne", Name: "one", value: 1}}},
			{TypeName: "Beta", Bits: []constInfo{{Const: "BetaOne", Name: "one", value: 1}}},
		}
	}

	t.Run("unqualified rejected", func(t *testing.T) {
		t.Parallel()

		if err := applyTrueValues(newSpecs(), []string{"AlphaOne"}); err == nil {
			t.Error("applyTrueValues unqualified with multiple types: expected error, got nil")
		}
	})

	t.Run("qualified assigns per type", func(t *testing.T) {
		t.Parallel()

		specs := newSpecs()
		if err := applyTrueValues(specs, []string{"Beta.BetaOne"}); err != nil {
			t.Fatalf("applyTrueValues qualified: unexpected error: %v", err)
		}

		if specs[0].TrueValue != "" {
			t.Errorf("Alpha.TrueValue = %q, want empty", specs[0].TrueValue)
		}

		if specs[1].TrueValue != "BetaOne" {
			t.Errorf("Beta.TrueValue = %q, want %q", specs[1].TrueValue, "BetaOne")
		}
	})
}
