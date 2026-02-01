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

func simplerangeL() {
	for from, to := range map[int]int{} {
		_, _ = from, to // want "is reachable"
	}
}

func simplerange2L(to int) {
	for from := range to { // want "is reachable"
		_ = from
	}
}

func simplerange3() {
	for to := range 5 {
		_ = to // want "unreachable"
		var from int
		_ = from
	}
}

func simplerange3L() {
	for to := range 5 {
		_ = to // want "is reachable"
		var from int
		_ = from
	}
}

func labeledRange(from, to int) int {
L:
	for range 1 {
		for range 5 {
			break L
		}

		return 0
	}

	return to // want "is reachable"
}
