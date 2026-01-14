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

package graph

import "log"

func simpleswitch() {
	from, to := 1, 5

	switch 5 {
	case 1:
		log.Fatal()

	case 3:
		return

	default:
		panic("")
	}

	_, _ = from, to // want "unreachable"
}

func simpleswitch2() {
	to := 5
	switch from := 1; from {
	case 1:
		if true {
			break
		}

		log.Fatal()

	case 3:
		return

	default:
		panic("")
	}

	_ = to // want "is reachable"
}

func simpleswitch3() {
	from, to := 1, 5

	switch from {
	case 1:
		if true {
			// break
		}

		log.Fatal()

	case to: // want "is reachable"
		return

	default:
		panic("")
	}
}

func simpleswitch4(from, to int) {
	switch from {
	case 1:
		if true {
			// break
		}

		log.Fatal()

	case 3:
		_ = to // want "is reachable"
		return

	default:
		panic("")
	}
}

func simpleswitch5() {
	from, to := 1, 5

	switch from {
	case 1:
		if true {
			// break
		}

		log.Fatal()

	case 3:
		log.Fatal()

		_ = to // want "unreachable"

	default:
		panic("")
	}
}

func fallthrough1() {
	to := 0
	switch {
	case true:
		from := 1
		_ = from

		fallthrough

	case false:
		_ = to // want "is reachable"
	}
}

func fallthrough2() {
	to := 0
	switch {
	case true:
		from := 1
		_ = from

		fallthrough

	default:
		_ = to // want "is reachable"
	}
}

func fallthrough3() {
	to := 0
	switch {
	default:
		from := 1
		_ = from

		fallthrough

	case false:
		_ = to // want "is reachable"
	}
}

func switchorder1() {
	to := 0
	switch {
	case func() bool { from := true; return from }():
		_ = to // want "is reachable"
	}
}

func switchorder2() {
	to := 0
	switch {
	case func() bool { from := false; return from }():

	case true:
		_ = to // want "is reachable"
	}
}

func switchorder3() {
	to := 0
	switch {
	case true:
		_ = to // want "unreachable"

	case func() bool { from := false; return from }():
	}
}
