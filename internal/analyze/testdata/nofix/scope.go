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

package nofix

import "fmt"

func skipScope() (b int) {
	var a any = 1

	b, ok := a.(int)
	if !ok {
		return
	}

	if b == 1 {
		fmt.Println("1")
	}

	c, ok := a.(string)
	if !ok {
		return
	}

	if c == "1" {
		fmt.Println("1")
	}

	return
}

func nextEven(i int) int {
	for j := i; ; {
		switch j % 2 {
		case 0:
			return j
		default:
			j++
		}
	}
}

func nestedScope() {
	x := 1
	if y := 1; y != 1 {
		fmt.Println(1)
	} else if z := 2; x != z {
		fmt.Println(2)
	}
}

func selectCase() {
	ch := make(chan int)
	var y int
	select {
	case x := <-ch:
		if true {
			fmt.Println(x)
		}
	case y = <-ch:
		if true {
			fmt.Println(y)
		}
	}
}

func caseClause() {
	ch := make(chan func())
	x := make(chan struct{})
	select {
	case ch <- func() {
		<-x
	}:
	}
}

func caseClause2() {
	ch := make(chan func())
	x := make(chan struct{})
	select {
	case ch <- func() {
		<-x
	}:
		x <- struct{}{}
	}
}
