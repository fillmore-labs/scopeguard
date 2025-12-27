// Copyright 2025-2026 Oliver Eikemeier. All Rights Reserved.
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

// Edge cases and complex scenarios

// Redeclared variable in x := x style
func redeclared() {
	for _, v := range []float64{1.0, -1.0} {
		v := math.Abs(v) // want "Variable 'v' can be moved to tighter if scope"
		if v > 0 {
			fmt.Println(v)
		}
	}
}

// Conflicting
func multiVar() {
	a := 1 // want "Variable 'a' can be moved to tighter if scope"
	b := 2 // want "Variable 'b' can be moved to tighter if scope"
	if a == 1 && b == 2 {
		fmt.Println("1, 2")
	}
}

// Variable used only in nested function literal - should NOT move into closure.
func nestedFunctionLiteral() {
	x := 1
	fn := func() {
		nested := func() {
			fmt.Println(x)
		}
		nested()
	}
	fn()
}

// Variable used in multiple levels of nesting.
func deeplyNested() {
	x := 1 // want "Variable 'x' can be moved to tighter block scope"
	{
		{
			{
				fmt.Println(x)
			}
		}
	}
}

// Variable used in both if and else, should move to if Init.
func ifElseInit() {
	val := compute() // want "Variable 'val' can be moved to tighter if scope"
	if val > 0 {
		fmt.Println("positive:", val)
	} else {
		fmt.Println("non-positive:", val)
	}
}

// Variable used in switch cases - should move to switch Init.
func switchCases() {
	val := compute() // want "Variable 'val' can be moved to tighter switch scope"
	switch val {
	case 1, 2, 3:
		fmt.Println("small:", val)
	default:
		fmt.Println("other:", val)
	}
}

// Variable used in range loop key/value - should NOT move.
func rangeKeyValue() {
	nums := []int{1, 2, 3}
	for i, v := range nums {
		fmt.Println(i, v)
	}
}

// Variable used in defer statement.
func deferStatement() {
	x := 1 // want "Variable 'x' can be moved to tighter block scope"
	{
		defer fmt.Println(x)
		fmt.Println("body")
	}
}

// Variable declared with blank identifier sibling.
func blankIdentifier() {
	x, _ := getTwo() // want "Variable 'x' can be moved to tighter block scope"
	{
		fmt.Println(x)
	}
}

// Multiple uses in same scope, different depths.
func sameScope() {
	x := 1 // want "Variable 'x' can be moved to tighter block scope"
	{
		if true {
			fmt.Println(x)
		}
		if false {
			fmt.Println(x)
		}
	}
}

// Variable used in select case expression.
func selectStatement() {
	ch, x := make(chan int), 1
	select {
	case ch <- x:
		fmt.Println("sent")
	default:
		fmt.Println("default")
	}
}

// Variable used in labeled loop statement.
func labeledStatement() {
	x := 1 // want "Variable 'x' can be moved to tighter if scope"
	if x == 1 {
	outer:
		for {
			if x > 1 {
				break outer
			}
			x++
		}
	}
}

// Variable used in go statement (spawning goroutine).
func goStatement() {
	x := 1
	go func() {
		fmt.Println(x) // Captured by closure
	}()
}

// Empty if body.
func emptyIfBody() {
	x := 1 // want "Variable 'x' can be moved to tighter if scope"
	if x == 1 {
	}
}

// Nested switches.
func nestedSwitches() {
	x := 1 // want "Variable 'x' can be moved to tighter switch scope"
	switch x {
	case 1:
		y := 2 // want "Variable 'y' can be moved to tighter switch scope"
		switch y {
		case 2:
			fmt.Println("nested")
		}
	}
}

// Move a little.
func notInitButBlock() {
	x := 1 // want "Variable 'x' can be moved to tighter block scope"
	{
		if y := 1; x == y {
			fmt.Println(x)
		}
	}
}

// Single var.
func simplevarStatement() {
	// var comment
	var x, y int = 1, func() int { return 2 }() // want "Variables 'x' and 'y' can be moved to tighter block scope"
	// post
	{
		fmt.Println(x, y)
	}
}

// Multiple vars. Note that a comment is lost (see also issue [#20744]).
//
// [#20744]: https://go.dev/issue/20744
func multivarStatement() {
	// var comment
	var ( // lost // want "Variables 'x' and 'y' can be moved to tighter block scope"
		// pre x
		x int = 1 // var x
		// pre y
		y int = func() int { return 2 }() // var y
	) // post
	{
		fmt.Println(x, y)
	}
}

// Redeclaration with a different type.
func redec() {
	type fooError = interface {
		error
		Foo() bool
	}

	e := func(i int) (int, error) { return i, nil }
	f := func(i int) (int, fooError) { return i, nil }

	a, err := e(1) // want "Variables 'a' and 'err' can be moved to tighter block scope"
	{
		_ = a
	}

	b, err := f(1)
	if err == nil {
		b = 0
	}
	_ = b

	c, err := e(1)
	_ = c
}

type T struct{ a int }

func (t T) value() int { return t.a }

// Composite Literal Call
func compositeLiteralCall() {
	x, y := 1, T{1}.value() // want "Variables 'x' and 'y' can be moved to tighter if scope"

	if x == 1 {
		fmt.Println(y)
	}
}

// Composite Literal Okay
func compositeLiteralOkay() {
	x, y := 1, func(t T) int { return t.value() }(T{1}) // want "Variables 'x' and 'y' can be moved to tighter if scope"

	if x == 1 {
		fmt.Println(y)
	}
}

// Composite Literal Bare - expression root is itself a composite literal
func compositeLiteralBare() {
	x, y := 1, T{1} // want "Variables 'x' and 'y' can be moved to tighter if scope"

	if x == 1 {
		fmt.Println(y.value())
	}
}

// Helper functions
func getTwo() (int, int) { return 1, 2 }
