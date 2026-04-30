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
	"reflect"
	"testing"

	. "fillmore-labs.com/scopeguard/internal/config"
)

func TestSafety_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		safety Safety
		want   string
	}{
		{"Unknown", Unknown, "unknown"},
		{"Safe", Safe, "safe"},
		{"Unsafe", Unsafe, "unsafe"},
		{"Breaking", Breaking, "breaking"},
		{"Invalid", Safety(1 << 3), "Safety(4)"},
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
		safety Safety
		want   string
	}{
		{"Safe", Safe, "safe"},
		{"Unsafe", Unsafe, "unsafe"},
		{"Breaking", Breaking, "breaking"},
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
		{"Unknown (error)", "unknown", 0, true},
		{"Invalid", "invalid", 0, true},
		{"Empty", "", 0, true},
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
		checks      map[Safety]bool
		name        string
		wantTiers   []string
		filter      SafetyFilter
		wantDefault bool
	}{
		{
			name:        "Default",
			filter:      FilterAll,
			wantDefault: true,
			wantTiers:   []string{"safe", "unsafe", "breaking"},
			checks: map[Safety]bool{
				Safe:     true,
				Unsafe:   true,
				Breaking: true,
			},
		},
		{
			name:        "Safe only",
			filter:      FilterSafe,
			wantDefault: false,
			wantTiers:   []string{"safe"},
			checks: map[Safety]bool{
				Safe:     true,
				Unsafe:   false,
				Breaking: false,
			},
		},
		{
			name:        "Safe and Breaking",
			filter:      FilterSafe | FilterBreaking,
			wantDefault: false,
			wantTiers:   []string{"safe", "breaking"},
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

			if got := tt.filter.Default(); got != tt.wantDefault {
				t.Errorf("Default() got %v, want %v", got, tt.wantDefault)
			}

			if got := tt.filter.Tiers(); !reflect.DeepEqual(got, tt.wantTiers) {
				t.Errorf("Tiers() got %v, want %v", got, tt.wantTiers)
			}

			for safety, wantEnabled := range tt.checks {
				if got := tt.filter.Allowed(safety); got != wantEnabled {
					t.Errorf("Enabled(%v) got %v, want %v", safety, got, wantEnabled)
				}
			}
		})
	}
}

func TestSafetyFilter_Set(t *testing.T) {
	t.Parallel()

	filter := FilterAll // all enabled by default

	if !filter.Allowed(Safe) || !filter.Allowed(Unsafe) || !filter.Allowed(Breaking) {
		t.Fatal("Expected all tiers to be enabled by default")
	}

	filter.Set(FilterUnsafe, false)

	if filter.Allowed(Unsafe) {
		t.Error("Expected Unsafe to be disabled after Set(FilterUnsafe, false)")
	}

	if !filter.Allowed(Safe) || !filter.Allowed(Breaking) {
		t.Error("Expected other tiers to remain enabled")
	}

	filter.Set(FilterUnsafe, true)

	if !filter.Allowed(Unsafe) {
		t.Error("Expected Unsafe to be enabled after Set(FilterUnsafe, true)")
	}

	filter.Set(FilterSafe|FilterBreaking, false)

	if filter.Allowed(Safe) || filter.Allowed(Breaking) {
		t.Error("Expected safe and breaking to be disabled")
	}
}
