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
	"sync"
)

func nestedReassigned() {
	var err error

	a, err := func() (int, error) {
		var a int
		if a, err = 1, error(nil); a != 0 { // want "Nested reassignment of variable 'err'"
			return a, err
		}

		return 0, nil
	}()

	fmt.Println(a, err)
}

func nestedFuncReassigned2() {
	x := 1
	x = x + func() int { x = 2; return x + 3 }() + x // want "Nested reassignment of variable 'x'"
	fmt.Println(x)
}

func nestedFuncReassigned() {
	var onceFunc func(int)

	onceFunc = func(i int) {
		onceFunc = func(int) {} // want "Nested reassignment of variable 'onceFunc'"

		fmt.Println(i)
	}

	onceFunc(1)
	onceFunc(2)
}

func noNestedFuncReassigned1() {
	var done bool
	onceFunc := func(i int) {
		if done {
			return
		}
		done = true

		fmt.Println(i)
	}

	onceFunc(1)
	onceFunc(2)
}

func noNestedFuncReassigned2() {
	var once sync.Once
	onceFunc := func(i int) {
		once.Do(func() {
			fmt.Println(i)
		})
	}

	onceFunc(1)
	onceFunc(2)
}
