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

func selectForeverL(from, to int) int {
	select {}

	return to // want "unreachable"
}

func selectAllL(from, to int) int {
	ch := make(chan int)
	select {
	case <-ch:
		return 0

	default:
		return 0
	}

	return to // want "unreachable"
}

func selectBreakL(from, to int) int {
	ch := make(chan int)
	select {
	case x := <-ch:
		if x > 0 {
			break
		}

		return 0
	}

	return to // want "is reachable"
}

func selectLabeledBreakL(to int) int {
	ch := make(chan int)

L:
	select {
	case x := <-ch:
		switch {
		case x > 0:
			from := 0
			_ = from

			fallthrough

		default:
			break L
		}

		return 0
	}

	return to // want "is reachable"
}

func selectToL() (to int) {
	for {
		ch := make(chan int)
		select {
		case from := <-ch:
			_ = from

		case to = <-ch: // want "is reachable"
			return
		}
	}
}

func selectTo() (to int) {
	for {
		ch := make(chan int)
		select {
		case from := <-ch:
			_ = from

		case to = <-ch: // want "unreachable"
			return
		}
	}
}

func selectEval() (to int) {
	select {
	default:
		_ = to // want "is reachable"
		return

	case <-func() chan int { var from int; _ = from; return make(chan int) }():
		return
	}
}
