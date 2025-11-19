// Copyright 2025 Oliver Eikemeier. All Rights Reserved.
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

package a

import (
	"fmt"
	"math"
)

// Simple case: variable declared early but only used in one block.
func simpleCase() {
	x := 1 // want "Variable 'x' can be moved to tighter block scope"
	if true {
		fmt.Println(x)
	}
}

// Simple var: changes call semantics.
func simpleVar() {
	var x float64 = math.Sin(math.Pi) // want "Variable 'x' can be moved to tighter block scope"
	if true {
		fmt.Println(x)
	}
}

// Variable used in loop post statement.
func loopPost() {
	i := 0 // want "Variable 'i' can be moved to tighter for scope"
	const m = 10
	for ; i < m; i++ {
		fmt.Println(i)
	}
}

// Variable used in multiple sibling blocks - CAN be moved to if Init.
func multipleBranches() {
	x := 1 // want "Variable 'x' can be moved to tighter if scope"
	if true {
		fmt.Println(x)
	} else {
		fmt.Println(x)
	}
}

// Variable used after the block - should NOT be moved.
func usedAfterBlock() {
	x := 1
	if true {
		fmt.Println(x)
	}
	fmt.Println(x)
}

// Variable declared at function scope, used only in nested block.
func nestedBlock() {
	x := 1 // want "Variable 'x' can be moved to tighter block scope"
	{
		fmt.Println(x)
	}
}

// Variable in loop - should not be moved inside the loop body.
func loopCase() {
	x := 1
	for i := 0; i < 10; i++ {
		fmt.Println(x, i)
	}
}

// Variable used in loop condition - CAN be moved to for Init.
func loopCondition() {
	x := true // want "Variable 'x' can be moved to tighter for scope"
	for x {
		x = false
	}
}

// Range loop - should not be moved inside the loop body.
func rangeLoop() {
	x := 1
	for i := range 5 {
		{
			fmt.Println(x, i)
		}
	}
}

// Function literal - should not be moved inside the function body.
func functionLiteral() {
	x := 1
	_ = func() int {
		{
			return x
		}
	}
}

// Select statement - should be moved inside the case.
func simplesSelectStatement(ch chan int, x int) {
	sent := "sent" // want "Variable 'sent' can be moved to tighter select case scope"
	select {
	case ch <- x:
		fmt.Println(sent)
	default:
		fmt.Println("default")
	}
}

// Functions
func functions() {
	even := func(i int) bool { return i%2 == 0 } // want "Variable 'even' can be moved to tighter if scope"

	if even(2) {
		fmt.Println(2)
	}
}

// Composite Literal
func compositeLiteral() {
	type T struct{ a int }

	x := &[...]T{{1}} // want "Variable 'x' can be moved to tighter if scope"

	if x == &([...]T{{1}}) {
		fmt.Println(x)
	}
}
