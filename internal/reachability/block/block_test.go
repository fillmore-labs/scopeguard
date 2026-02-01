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

package block_test

import (
	"go/token"
	"testing"

	. "fillmore-labs.com/scopeguard/internal/reachability/block"
)

func TestBlockFactory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		count int
	}{
		{"Empty", 0},
		{"Single", 1},
		{"BlockSize", ChunkSize},
		{"BlockSizePlusOne", ChunkSize + 1},
		{"MultiplePages", 2*ChunkSize + 46},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var f Factory

			for i := range tt.count {
				b := f.New(token.Pos(i + 1))
				b.End = token.Pos(i + 2)
			}

			blocks := f.All()
			if got, want := len(blocks), tt.count; got != want {
				t.Errorf("Got %d blocks, expected %d", got, want)
			}

			for i, b := range blocks {
				if got, want := b.Pos, token.Pos(i+1); got != want {
					t.Errorf("Got start position %d for block %d, expected %d", got, i, want)
				}
			}
		})
	}
}
