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

package graph

import (
	"context"
	"go/ast"
	"go/token"
	"go/types"
	"runtime/trace"

	"fillmore-labs.com/scopeguard/internal/reachability/block"
	"fillmore-labs.com/scopeguard/internal/reachability/tracker"
)

// BlockInterval represents a range in the source file with successor block indices for control-flow analysis.
type BlockInterval struct {
	Start, End token.Pos // The range of the block in the source file.
	Successors []int     // Indices of successor blocks in the intervals slice.
}

// Compare returns whether the position p is within the interval, before or after.
func (bi BlockInterval) Compare(p token.Pos) int {
	switch {
	case bi.End <= p:
		return -1

	case bi.Start > p:
		return 1

	default:
		return 0
	}
}

// BuildGraph constructs control-flow graph intervals for the given function body.
func BuildGraph(ctx context.Context, info *types.Info, recv *ast.FieldList, typ *ast.FuncType, body *ast.BlockStmt, forwardOnly bool) []BlockInterval {
	if body == nil {
		return nil
	}

	defer trace.StartRegion(ctx, "Graph").End()

	blocks := traverseFunc(info, recv, typ, body, forwardOnly)

	return buildIntervals(blocks)
}

func traverseFunc(info *types.Info, recv *ast.FieldList, typ *ast.FuncType, body *ast.BlockStmt, forwardOnly bool) []*block.Block {
	b := builder{
		labels:      make(map[string]*LabelTarget),
		forwardOnly: forwardOnly,
		Tracker: tracker.New(
			info,
		),
	}

	fun := b.New(typ.Pos()) // function literal

	fun.AddFields(recv)
	fun.AddFields(typ.Params)
	fun.AddFields(typ.Results)

	_ = b.appendStmtList(fun, body.List)

	return b.All()
}

// buildIntervals creates a list of block intervals from the CFG blocks.
func buildIntervals(blocks []*block.Block) []BlockInterval {
	// Build index map: maps each block to its position in the sorted slice
	idxMap := make(map[*block.Block]int, len(blocks))
	for i, block := range blocks {
		idxMap[block] = i
	}

	// Reusable set for tracking visited blocks during recursive successor traversal
	seen := make(map[*block.Block]struct{}, len(blocks))

	// Build intervals
	intervals := make([]BlockInterval, len(blocks))
	for i, block := range blocks {
		successors := make([]int, 0, 2)

		successors = appendSuccessors(successors, block, idxMap, seen)
		intervals[i] = BlockInterval{
			Start:      block.Pos,
			End:        block.End,
			Successors: successors,
		}

		clear(seen) // Reset the seen set for the next iteration
	}

	return intervals
}

// appendSuccessors recursively collects successor block indices.
func appendSuccessors(successors []int, b *block.Block, idxMap map[*block.Block]int, seen map[*block.Block]struct{}) []int {
	for _, succ := range [...]*block.Block{b.Successor1, b.Successor2} {
		if succ == nil {
			continue
		}

		if _, ok := seen[succ]; ok { // prevent infinite recursion
			continue
		}
		seen[succ] = struct{}{}

		idx, ok := idxMap[succ]
		if !ok {
			// This successor in an empty block and was removed from idxMap.
			// Recursively flatten it by including its successors instead.
			successors = appendSuccessors(successors, succ, idxMap, seen)

			continue
		}

		successors = append(successors, idx)
	}

	return successors
}
