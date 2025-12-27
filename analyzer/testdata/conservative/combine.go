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

package conservative

import "fmt"

func combine() {
	x := 1
	y := 2
	if x < y {
		fmt.Println(x, y)
	}
}

func reassigned() {
	x := 1
	px := &x
	z := *px
	_, x, y := 1, 2, 3

	if y > z {
		println(z) // would print 2 when z skips to initialization
	}

	_ = x
}

func reassigned2() {
	x := 1
	px := &x
	_, x, y := 1, 2, 3
	z := *px

	if y > z {
		println(z) // would print 1 when combined
	}
}
