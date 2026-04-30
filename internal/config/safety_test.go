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

package config_test

import (
	"testing"

	. "fillmore-labs.com/scopeguard/internal/config"
)

func TestSafety_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		want   string
		safety Safety
	}{
		{"Nothing", "none", Nothing},
		{"Safe", "safe", Safe},
		{"Unsafe", "unsafe", Unsafe},
		{"Breaking", "breaking", Breaking},
		{"Invalid", "Safety: invalid value 0x8", Safety(1 << 3)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.safety.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSafety_MarshalText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		want   string
		safety Safety
	}{
		{"Safe", "safe", Safe},
		{"Unsafe", "unsafe", Unsafe},
		{"Breaking", "breaking", Breaking},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b, err := tt.safety.MarshalText()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got := string(b); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSafety_UnmarshalText(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		text    string
		want    Safety
		wantErr bool
	}{
		{"Safe", "safe", Safe, false},
		{"Unsafe", "unsafe", Unsafe, false},
		{"Breaking", "breaking", Breaking, false},
		{"Invalid", "invalid", Nothing, true},
		{"Empty", "", Nothing, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got Safety

			if err := got.UnmarshalText([]byte(tt.text)); (err != nil) != tt.wantErr {
				t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafetyFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		checks map[Safety]bool
		name   string
		tiers  string
		filter Safety
	}{
		{
			name:   "Default",
			filter: All,
			tiers:  "all",
			checks: map[Safety]bool{
				Safe:     true,
				Unsafe:   true,
				Breaking: true,
			},
		},
		{
			name:   "Safe only",
			filter: Safe,
			tiers:  "safe",
			checks: map[Safety]bool{
				Safe:     true,
				Unsafe:   false,
				Breaking: false,
			},
		},
		{
			name:   "Safe and Breaking",
			filter: Safe | Breaking,
			tiers:  "safe, breaking",
			checks: map[Safety]bool{
				Safe:     true,
				Unsafe:   false,
				Breaking: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.filter.String(); got != tt.tiers {
				t.Errorf("Tiers() got %s, want %s", got, tt.tiers)
			}

			for safety, wantEnabled := range tt.checks {
				if got := tt.filter.Has(safety); got != wantEnabled {
					t.Errorf("Enabled(%v) got %v, want %v", safety, got, wantEnabled)
				}
			}
		})
	}
}

func TestSafetyFilter_Set(t *testing.T) {
	t.Parallel()

	filter := All // all enabled by default

	if !filter.Has(Safe) || !filter.Has(Unsafe) || !filter.Has(Breaking) {
		t.Fatal("Expected all tiers to be enabled by default")
	}

	filter.Set(Unsafe, false)

	if filter.Has(Unsafe) {
		t.Error("Expected Unsafe to be disabled after Set(Unsafe, false)")
	}

	if !filter.Has(Safe) || !filter.Has(Breaking) {
		t.Error("Expected other tiers to remain enabled")
	}

	filter.Set(Unsafe, true)

	if !filter.Has(Unsafe) {
		t.Error("Expected Unsafe to be enabled after Set(Unsafe, true)")
	}

	filter.Set(Safe|Breaking, false)

	if filter.Has(Safe) || filter.Has(Breaking) {
		t.Error("Expected safe and breaking to be disabled")
	}
}
