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

package b

import "fmt"

func alreadyDeclared() {
	x := 1 // want "Variable 'x' can be moved to tighter block scope"
	if true {
		x := fmt.Sprintf("%d", x)
		fmt.Println(x)
	}
}

func fixBreaks() {
	a, ok := 1, true // want "Variables 'a' and 'ok' can be moved to tighter if scope"
	if ok {
		fmt.Println(a)
	}

	b, ok := 2, false // want "Variable 'ok' is unused and can be removed"
	fmt.Println(b)
}

type A int

func (a A) int() int { return int(a) }

func moveOver() {
	var a, b A // want "Variables 'a' and 'b' can be moved to tighter block scope"

	a, c := 3, 4

	fmt.Println(a.int())

	if true {
		b = 6

		fmt.Println(a, b, c)
	}
}

type myError int

func (m myError) Error() string { return fmt.Sprintf("error %d", m) }

func moveOver2() {
	var a, b error // want "Variables 'a' and 'b' can be moved to tighter block scope"

	a, c := func() (myError, myError) { return 1, 2 }()

	a = func() error { return nil }()

	if true {
		b = func() myError { return 3 }()

		fmt.Println(a, b, c)
	}
}
