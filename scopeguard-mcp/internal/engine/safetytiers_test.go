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

package engine_test

import (
	"encoding/json"
	"testing"

	"fillmore-labs.com/scopeguard/internal/config"
	. "fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
)

func TestSafetyTiers_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		want   string
		filter config.SafetyFilter
	}{
		{
			name:   "Empty",
			filter: config.FilterNothing,
			want:   `[]`,
		},
		{
			name:   "Safe only",
			filter: config.FilterSafe,
			want:   `["safe"]`,
		},
		{
			name:   "Unsafe only",
			filter: config.FilterUnsafe,
			want:   `["unsafe"]`,
		},
		{
			name:   "Breaking only",
			filter: config.FilterBreaking,
			want:   `["breaking"]`,
		},
		{
			name:   "Safe and Breaking",
			filter: config.FilterSafe | config.FilterBreaking,
			want:   `["safe","breaking"]`,
		},
		{
			name:   "All tiers",
			filter: config.FilterAll,
			want:   `["safe","unsafe","breaking"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b, err := json.Marshal(SafetyTiers(tt.filter))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got := string(b); got != tt.want {
				t.Errorf("MarshalJSON() got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSafetyTiers_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    config.SafetyFilter
		wantErr bool
	}{
		{
			name:  "Empty array",
			input: `[]`,
			want:  config.FilterNothing,
		},
		{
			name:  "Null",
			input: `null`,
			want:  config.FilterAll,
		},
		{
			name:  "Safe only",
			input: `["safe"]`,
			want:  config.FilterSafe,
		},
		{
			name:  "Safe and Breaking",
			input: `["breaking", "safe"]`,
			want:  config.FilterSafe | config.FilterBreaking,
		},
		{
			name:  "All tiers",
			input: `["safe", "unsafe", "breaking"]`,
			want:  config.FilterAll,
		},
		{
			name:    "Invalid format",
			input:   `{}` + "\n",
			wantErr: true,
		},
		{
			name:    "Invalid value",
			input:   `["invalid"]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got SafetyTiers

			if err := json.Unmarshal([]byte(tt.input), &got); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error got %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != SafetyTiers(tt.want) {
				t.Errorf("UnmarshalJSON() got %v, want %v", config.SafetyFilter(got).Tiers(), tt.want.Tiers())
			}
		})
	}
}

func TestSafetyTiers_RoundTrip(t *testing.T) {
	t.Parallel()

	filters := []config.SafetyFilter{
		config.FilterNothing,
		config.FilterSafe,
		config.FilterUnsafe,
		config.FilterBreaking,
		config.FilterSafe | config.FilterUnsafe,
		config.FilterSafe | config.FilterBreaking,
		config.FilterUnsafe | config.FilterBreaking,
		config.FilterAll,
	}

	for _, f := range filters {
		original := SafetyTiers(f)

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal(%v) error: %v", f, err)
		}

		var decoded SafetyTiers
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal(%s) error: %v", string(data), err)
		}

		if decoded != original {
			t.Errorf("round trip got %v, want %v (json: %s)", decoded, original, string(data))
		}
	}
}

func TestSafetyTiers_Filter(t *testing.T) {
	t.Parallel()

	cases := []config.SafetyFilter{
		config.FilterNothing,
		config.FilterSafe,
		config.FilterAll,
	}

	for _, want := range cases {
		filter := SafetyTiers(want)
		if got := filter.Filter(); got != want {
			t.Errorf("Filter() got %v, want %v", got, want)
		}
	}

	if got := (*SafetyTiers)(nil).Filter(); got != config.FilterAll {
		t.Errorf("nil.Filter() got %v, want FilterAll", got)
	}
}

// TestSafetyTiers_PointerOmitEmpty confirms that a *SafetyTiers field with
// `json:",omitempty"` is omitted when nil and rendered when present. This is
// the contract callers rely on for "default → include all tiers".
func TestSafetyTiers_PointerOmitEmpty(t *testing.T) {
	t.Parallel()

	type wrapper struct {
		Safety *SafetyTiers `json:"safety,omitempty"`
	}

	bNil, err := json.Marshal(wrapper{})
	if err != nil {
		t.Fatalf("Marshal(nil) error: %v", err)
	}

	if got, want := string(bNil), `{}`; got != want {
		t.Errorf("Marshal(nil) got %s, want %s", got, want)
	}

	tiers := SafetyTiers(config.FilterSafe)

	bSet, err := json.Marshal(wrapper{Safety: &tiers})
	if err != nil {
		t.Fatalf("Marshal(&FilterSafe) error: %v", err)
	}

	if got, want := string(bSet), `{"safety":["safe"]}`; got != want {
		t.Errorf("Marshal(&FilterSafe) got %s, want %s", got, want)
	}

	var decoded wrapper
	if err := json.Unmarshal([]byte(`{"safety":null}`), &decoded); err != nil {
		t.Fatalf("Unmarshal(null) error: %v", err)
	}

	if decoded.Safety != nil {
		t.Errorf("Unmarshal(null) got %v, want nil", decoded.Safety)
	}
}
