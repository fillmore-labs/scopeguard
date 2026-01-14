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

package block

import (
	"go/token"
	"slices"
)

// Factory creates and manages [Block]s in a [slab list].
//
// [slab list]: https://en.wikipedia.org/wiki/Slab_allocation
type Factory struct {
	start, current *chunk
	count, total   int
}

// chunk is a linked list of fixed-size arrays of Blocks.
type chunk struct {
	blocks [chunkSize]Block
	next   *chunk
}

// chunkSize defines the number of Blocks stored in a single chunk.
const chunkSize = 127

// New creates and returns a new *[Block] and adds it to the list of existing blocks.
func (f *Factory) New(pos token.Pos) *Block {
	if f.count == chunkSize {
		f.current.next = new(chunk)
		f.current = f.current.next
		f.count = 0
		f.total += chunkSize
	} else if f.current == nil {
		f.current = new(chunk)
		f.start = f.current
	}

	f.count++

	block := &f.current.blocks[f.count-1]
	block.Pos = pos

	return block
}

// All retrieves all non-empty Blocks managed by the BlockFactory in source order.
func (f *Factory) All() []*Block {
	if f.count == 0 {
		return nil
	}

	blocks := make([]*Block, 0, f.count+f.total)
	for next := f.start; next != nil; next = next.next {
		n := chunkSize
		if next == f.current {
			n = f.count
		}

		for i := range n {
			block := &next.blocks[i]
			if block.isEmpty() {
				continue
			}

			blocks = append(blocks, block)
		}
	}

	// Sort by source order
	slices.SortFunc(blocks, (*Block).cmp)

	return blocks
}
