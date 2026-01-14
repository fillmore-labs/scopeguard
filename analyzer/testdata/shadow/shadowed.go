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

package shadow

import (
	"fmt"
	"log"
	"strings"
)

func shadowed() {
	i, a := -1, true

	if a {
		i := -i
		fmt.Println(i)
	}

	i, b := i-1, true // want "Variable 'i' used after previously shadowed"

	if b {
		var i int = -i
		fmt.Println(i)
	}

	i, c := i-1, true // want "Variable 'i' used after previously shadowed"

	if c {
		var i int = -i
		fmt.Println(i)
	}

	i, d := i-1, true // want "Variable 'i' used after previously shadowed"

	if d {
		i := -i
		fmt.Println(i)
	}

	i = i - 1 // want "Variable 'i' used after previously shadowed"
	i, e := i-1, true

	if e {
		var i int = -i
		fmt.Println(i)
	}

	fmt.Println(i) // want "Variable 'i' used after previously shadowed"
}

func shadowedReturn() (i int) {
	i, a := -1, true

	if a {
		i := -i
		fmt.Println(i)
	}

	return // want "Variable 'i' used after previously shadowed"
}

func shadowedReturnUnreachable() (i int) {
	i, a := -1, true

	if a {
		i := -i
		return i
	}

	return
}

func notReachable() (err error) {
	var sb strings.Builder

	if _, err := sb.WriteString(""); err != nil {
		panic(err)
	} else {
		log.Fatal(err)
	}

	return
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
		fmt.Println(a) // want "Variable 'a' used after previously shadowed"

	case 3:
		a := a + 1
		fmt.Println(a)

	default:
		fmt.Println(a)
	}

	fmt.Println(a) // want "Variable 'a' used after previously shadowed"
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
		fmt.Println(a) // want "Variable 'a' used after previously shadowed"

	case a := <-ch:
		_ = a

	case ch <- a:
		fmt.Println(a)
	}

	fmt.Println(a) // want "Variable 'a' used after previously shadowed"
}

func elseAssign() error {
	var err error
	if true {
		var err error

		_ = err
	} else {
		err = nil
	}

	return err // want "Variable 'err' used after previously shadowed"
}

func ifelseuse() {
	f := func() (int, error) { return 0, nil }

	if x, err := f(); err == nil {
		fmt.Println(x)
		if x, err := f(); err == nil {
			fmt.Println(x)
		}
	} else {
		fmt.Println(err)
	}
}

func nestedAssignment() int {
	i := 1
	{
		i := 2
		_ = i
	}

	i = func() int {
		i = 3
		return i
	}()

	return i
}

func reassigned() {
	i := 1

	{
		i := 2
		_ = i
	}

	if true {
		i = 3
	} else {
		_ = i // want "Variable 'i' used after previously shadowed"
	}
	i = i + 1

	{
		i := 4
		_ = i
	}

	i = i + 1 // want "Variable 'i' used after previously shadowed"

	{
		i := 5
		_ = i
	}

	i++ // want "Variable 'i' used after previously shadowed"
}

func typeSwitch() {
	var a any

	var err error

	switch err := a.(type) {
	case error:

	default:
		_ = err
	}

	_ = err // want "Variable 'err' used after previously shadowed"
}

func typeSwitchOk() {
	var a any

	switch a := a.(type) {
	case int:

	default:
		_ = a
	}

	_ = a
}
