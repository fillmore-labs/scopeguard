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

package reachability

import (
	"context"
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"fillmore-labs.com/scopeguard/internal/reachability/graph"
)

// Graph determines reachability in a control-flow graph.
type Graph struct {
	// Lazy evaluation: graph construction is deferred until first Reachable() call
	buildIntervals func() []graph.BlockInterval

	// Intervals, strictly sorted by start position for binary search
	intervals []graph.BlockInterval

	// Reusable BFS state to avoid allocations on each reachability check
	seen  []bool // Visited set
	queue []int  // Ring buffer
}

// NewGraph analyzes control flow to determine reachability between nodes.
// forwardOnly enforces forward-only reachability.
func NewGraph(ctx context.Context, info *types.Info, recv *ast.FieldList, typ *ast.FuncType, body *ast.BlockStmt, forwardOnly bool) *Graph {
	buildIntervals := func() []graph.BlockInterval {
		return graph.BuildGraph(ctx, info, recv, typ, body, forwardOnly)
	}

	return &Graph{buildIntervals: buildIntervals}
}

// Reachable determines if the position `to` is reachable from the position `from`.
// It initializes the graph if it hasn't been built yet.
func (g *Graph) Reachable(from, to token.Pos) (reachable, ok bool) {
	if g == nil {
		return true, false
	}

	if g.intervals == nil {
		g.init()
	}

	return g.reachable(from, to)
}

func (g *Graph) init() {
	g.intervals = g.buildIntervals()

	// Allocate reusable BFS state sized to the number of blocks.
	// These are reset on each reachability check rather than reallocated.
	g.queue = make([]int, len(g.intervals))
	g.seen = make([]bool, len(g.intervals))
}

// reachable performs the actual reachability check using cached intervals.
func (g *Graph) reachable(from, to token.Pos) (reachable, ok bool) {
	source, ok := g.indexOf(from)
	if !ok {
		return true, false
	}

	target, ok := g.indexOf(to)
	if !ok {
		return true, false
	}

	// Are we at a later position in the same block?
	if source == target && to >= from {
		return true, true
	}

	clear(g.seen) // Reset visited set from previous checks

	// We use a ring buffer queue to minimize allocations.
	qTail := g.enqueueSuccessors(source, 0)

	// Determine reachability using BFS.
	for qHead := 0; qHead < qTail; qHead++ {
		curr := g.queue[qHead]

		if curr == target {
			return true, true
		}

		qTail = g.enqueueSuccessors(curr, qTail)
	}

	return false, true
}

// enqueueSuccessors adds unseen successors of block s to the queue.
func (g *Graph) enqueueSuccessors(s, qTail int) int {
	for _, succ := range g.intervals[s].Successors {
		if g.seen[succ] {
			continue
		}
		g.seen[succ] = true

		g.queue[qTail] = succ
		qTail++
	}

	return qTail
}

func (g *Graph) indexOf(pos token.Pos) (int, bool) {
	return slices.BinarySearchFunc(g.intervals, pos, graph.BlockInterval.Compare)
}
