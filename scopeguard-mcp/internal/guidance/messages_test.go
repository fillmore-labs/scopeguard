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

package guidance_test

import (
	"regexp"
	"testing"

	"fillmore-labs.com/scopeguard/internal/config"
	"fillmore-labs.com/scopeguard/scopeguard-mcp/internal/engine"
	. "fillmore-labs.com/scopeguard/scopeguard-mcp/internal/guidance"
)

func TestFilterPhrase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		f    engine.Facts
		want string
	}{
		{
			name: "default tiers",
			f:    engine.Facts{Filter: config.All},
			want: "",
		},
		{
			name: "safe only",
			f:    engine.Facts{Filter: config.Safe},
			want: `\(safety filters: safe\)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := FilterPhrase(tt.f); got != tt.want {
				if tt.want == "" || !regexp.MustCompile(tt.want).MatchString(got) {
					t.Errorf("FilterPhrase() = %v, want matching %v", got, tt.want)
				}
			}
		})
	}
}

func TestTruncationPhrase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		f     engine.Facts
		topic string
		want  string
	}{
		{
			name:  "no dropped",
			f:     engine.Facts{Counts: engine.Counts{Total: 0}},
			topic: "limits",
			want:  "",
		},
		{
			name:  "dropped items",
			f:     engine.Facts{Counts: engine.Counts{Total: 5, Dropped: 5}},
			topic: "limits",
			want:  `5 not shown \(see help topic 'limits'\)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := TruncationPhrase(tt.f, tt.topic); got != tt.want {
				if tt.want == "" || !regexp.MustCompile(tt.want).MatchString(got) {
					t.Errorf("TruncationPhrase() = %v, want matching %v", got, tt.want)
				}
			}
		})
	}
}

func TestHelpRef(t *testing.T) {
	t.Parallel()

	if got, want := HelpRef("test_topic"), `see help topic 'test_topic'`; got != want {
		t.Errorf("HelpRef() = %v, want %v", got, want)
	}
}

func TestJoin(t *testing.T) {
	t.Parallel()

	if got, want := Join("; ", "a", "", "b", "c"), "a; b; c"; got != want {
		t.Errorf("Join() = %v, want %v", got, want)
	}
}
