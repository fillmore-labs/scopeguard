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
	"strconv"
)

// Redefinition - would be broken by fix
func goStringer() {
	var b fmt.Stringer = B{"test"}
	s := b.String() // want "Variable 's' can be moved to tighter if scope"

	if b, ok := b.(fmt.GoStringer); ok {
		if s == b.GoString() {
			fmt.Println("equal")
		}
	}
}

type I int

func (i I) String() string { return strconv.Itoa(int(i)) }

const two = 2

// Inherited type - would be broken by fix
func intString() {
	i, j := I(4), 4 // want "Variables 'i' and 'j' can be moved to tighter block scope"
	i, k := two, 2

	if k != 0 {
		fmt.Println(i, j)
	}

	fmt.Println(i.String(), k)
}

// Untypes nil use - would be broken by fix
func untypedNilUse() {
	var err error

	err, ptr := nil, *new(error) // want "Variables 'err' and 'ptr' can be moved to tighter if scope"
	if err == ptr {
		fmt.Println(err)
	}
}

// Crossing labeled statement
func crossesLabel() {
	i := 0
	j := i
label:
	{
		if i < 1 || i <= j {
			i++
			goto label
		}
	}
}
