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

package check

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"golang.org/x/tools/go/cfg"
)

// CFG determines reachability in a control-flow graph.
type CFG struct {
	intervals []blockInterval
}

// NewCFG creates a new [CFG] from the given *[ast.BlockStmt].
func NewCFG(info *types.Info, body *ast.BlockStmt) CFG {
	cfg := cfg.New(body, func(expr *ast.CallExpr) bool {
		return canReturn(info, expr)
	})

	return CFGFromCFG(cfg)
}

// CFGFromCFG converts a control-flow graph [cfg.CFG] into a [CFG] for efficient reachability checks.
func CFGFromCFG(cfg *cfg.CFG) CFG {
	intervals := sortedBlocks(cfg.Blocks)

	return CFG{intervals: intervals}
}

// Reachable reports whether the position `to` is reachable from the position `from`
// in the control-flow graph.
func (c CFG) Reachable(from, to token.Pos) (reachable, ok bool) {
	start, finish := c.blockFor(from), c.blockFor(to)
	if start == nil || finish == nil {
		return true, false
	}

	// Are we at a later position in the same block?
	if start == finish && to >= from {
		return true, true
	}

	// Determine reachability using BFS.
	// We use a ring buffer queue to minimize allocations.
	qHead, qTail := 0, 0
	queue := make([]*cfg.Block, len(c.intervals)) // Upper bound for queue size is number of blocks

	// Track seen blocks to prevent duplicates in the queue.
	// This ensures the queue size never exceeds the number of blocks.
	seen := make(map[*cfg.Block]struct{})

	// Enqueue initial successors
	qTail = addSuccessors(start, qTail, queue, seen)

	for qHead < qTail {
		block := queue[qHead]

		qHead++

		if block == finish {
			return true, true
		}

		qTail = addSuccessors(block, qTail, queue, seen)
	}

	return false, true
}

func addSuccessors(block *cfg.Block, qTail int, queue []*cfg.Block, seen map[*cfg.Block]struct{}) int {
	for _, succ := range block.Succs {
		if _, ok := seen[succ]; ok {
			continue
		}
		seen[succ] = struct{}{}

		// If the successor block is empty (no nodes), it doesn't represent a code region we care about finding,
		// so we skip it and process its successors immediately. This flattens the graph for our purposes.
		if len(succ.Nodes) == 0 {
			qTail = addSuccessors(succ, qTail, queue, seen)
			continue
		}

		queue[qTail] = succ
		qTail++
	}

	return qTail
}

func (c CFG) blockFor(pos token.Pos) *cfg.Block {
	idx, ok := slices.BinarySearchFunc(c.intervals, pos, blockInterval.compare)
	if !ok {
		return nil
	}

	return c.intervals[idx].block
}

func sortedBlocks(blocks []*cfg.Block) []blockInterval {
	intervals := make([]blockInterval, 0, len(blocks))

	for _, block := range blocks {
		l := len(block.Nodes)
		if l == 0 {
			continue
		}

		// We use the range of the first node to the last node.
		// Note: This relies on nodes within a basic block being contiguous in source order.
		intervals = append(intervals, blockInterval{
			start: block.Nodes[0].Pos(),
			end:   block.Nodes[l-1].End(),
			block: block,
		})
	}

	// Sort intervals by start position to enable binary search.
	slices.SortFunc(intervals, blockInterval.sort)

	return intervals
}

type blockInterval struct {
	start, end token.Pos
	block      *cfg.Block
}

func (bi blockInterval) sort(bo blockInterval) int {
	return int(bi.start - bo.start)
}

func (bi blockInterval) compare(p token.Pos) int {
	switch {
	case bi.end < p:
		return -1

	case bi.start > p:
		return 1

	default:
		return 0
	}
}
