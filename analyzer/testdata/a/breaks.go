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
	"errors"
	"fmt"
	"path"
	"strings"
)

type B struct{ s string }

func (b B) String() string { return b.s }

func (b B) GoString() string { return b.s }

// Redefinition - fine
func redef2() {
	var b fmt.Stringer = B{"test"}
	s := b.String() // want "Variable 's' can be moved to tighter block scope"

	if true {
		_ = s
		b, ok := b.(fmt.GoStringer)
		if !ok {
			return
		}
		if s == b.GoString() {
			fmt.Println("equal")
		}
	}
}

// Shadowing - fix breaks
func shadow(name string) []string {
	dir := path.Dir(name)   // want "Variable 'dir' can be moved to tighter if scope"
	base := path.Base(name) // want "Variable 'base' can be moved to tighter if scope"

	var dirs []string
	if dir != "." {
		dirs = append(dirs, strings.Split(dir, "/")...)
	}
	path := dirs
	if base != "." {
		path = append(path, "/", base)
	}

	return path
}

// Unused
func unused() {
	a, err := 1, errors.New("test") // want "Variables 'a' and 'err' can be moved to tighter if scope"
	if err != nil {
		fmt.Println(a)
	}

	b, err := 1, errors.New("test") // want "Variable 'b' can be moved to tighter if scope"
	if b != 0 {
		fmt.Println(b)
	}
}

func unusedErr() {
	err := func() error { return nil }() // want "Variable 'err' is unused and can be removed"

	a, err := func() (int, error) { return 0, nil }() // want "Variables 'a' and 'err' can be moved to tighter if scope"

	if a == 0 {
		fmt.Println(a, err)
	}
}

func alreadyDeclared() {
	x := 1 // want "Variable 'x' can be moved to tighter block scope"
	if true {
		x := fmt.Sprintf("%d", x)
		fmt.Println(x)
	}
}

func madeUnused() {
	a, ok := 1, true // want "Variables 'a' and 'ok' can be moved to tighter if scope"
	if ok {
		fmt.Println(a)
	}

	b, ok := 2, false // want "Variable 'ok' is unused and can be removed"
	fmt.Println(b)
}
