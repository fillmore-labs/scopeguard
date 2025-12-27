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

import "fmt"

func shadowed() {
	i, a := -1, true

	if a {
		i := -i
		fmt.Println(i)
	}

	i, b := i-1, true // want "Identifier 'i' used after previously shadowed"

	if b {
		i := -i
		fmt.Println(i)
	}

	i, c := i-1, true // want "Identifier 'i' used after previously shadowed"

	if c {
		var i int = -i
		fmt.Println(i)
	}

	i, d := i-1, true // want "Identifier 'i' used after previously shadowed"

	if d {
		i := -i
		fmt.Println(i)
	}

	i = i - 1 // want "Identifier 'i' used after previously shadowed"
	i, e := i-1, true

	if e {
		var i int = -i
		fmt.Println(i)
	}

	fmt.Println(i) // want "Identifier 'i' used after previously shadowed"
}

func shadowedReturn() (i int) {
	i, a := -1, true

	if a {
		i := -i
		return i
	}

	return // want "Identifier 'i' used after previously shadowed"
}

func shadowedFunc() {
	var err error

	a, err := func() (int, error) {
		a, err := 1, error(nil)

		return a, err
	}()

	fmt.Println(a, err)
}

func shadowedFuncNested() {
	var err error

	a, err := func() (int, error) {
		if a, err := 1, error(nil); a != 0 {
			return a, err
		}

		return 0, nil
	}()

	fmt.Println(a, err)
}

func reassignedFuncNested() {
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

func cases() {
	a := 1

	switch a {
	case 1:
		a := a + 1
		fmt.Println(a)

	case 2:
		{
			a := a + 1
			_ = a
		}
		fmt.Println(a) // This might be confusing

	case 3:
		a := a + 1
		fmt.Println(a)

	default:
		fmt.Println(a)
	}

	fmt.Println(a) // want "Identifier 'a' used after previously shadowed"
}

func sends() {
	a := 1

	ch := make(chan int)
	select {
	case a := <-ch:
		{
			a := a + 1
			_ = a
		}
		fmt.Println(a) // want "Identifier 'a' used after previously shadowed"

	case ch <- a: // want "Identifier 'a' used after previously shadowed"
		fmt.Println(a)
	}

	fmt.Println(a)
}
