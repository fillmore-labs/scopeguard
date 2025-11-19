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

import "fmt"

// Same-type redeclarations - first declaration should be removable
func sameType() {
	var x int // want "Variable 'x' is unused and can be removed"

	x, y := 1, 2
	if y > 0 {
		x, z := 3.0, 4.0
		fmt.Println(x, z)
	}

	fmt.Println(x)
}

// Interface to concrete type (reverse direction) - should be removable
func interfaceToConcreteReverse() {
	type fooError = interface {
		error
		Foo() bool
	}

	var err error // want "Variable 'err' is unused and can be removed"

	a, err := func() (int, error) {
		return 1, nil
	}()
	if err != nil {
		return
	}

	b, err := func() (int, fooError) {
		return 2, nil
	}()
	if err != nil {
		return
	}

	fmt.Println(a, b)
}

// Multi-variable declaration where one needs to be kept
// The declaration "var x, y int" has x unused, but y has type-incompatible redeclarations.
// We must keep the entire declaration because of y.
func multiVarDecl() {
	type fooError = interface {
		error
		Foo() bool
	}

	var x, y error // x is unused in first decl, but y needs the type // want "Variables 'x' and 'y' are unused and can be removed"

	// x is used later with same type (safe to remove x from first decl)
	a, x := func() (int, error) { return 1, nil }()
	fmt.Println(a, x)

	// y has type-incompatible redeclarations (must keep first decl)
	y, z := func() (fooError, int) { return nil, 1 }()
	fmt.Println(y, z)

	y, w := func() (error, int) { return nil, 2 }()
	fmt.Println(y, w)
}
