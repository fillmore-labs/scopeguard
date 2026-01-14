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

func basic() {
	var to int

	if true {
		var from int
		_ = from
	}

	_ = to // want "is reachable"
}

func simple2() {
	var to int
	_ = to // want "unreachable"

	if true {
		var from int
		_ = from
	}
}

func loop(to int) {
	for {
		_ = to // want "unreachable"

		var from int
		_ = from
	}
}

func loopL() {
	for {
		var to int
		_ = to // want "is reachable"

		var from int
		_ = from
	}
}

func infiniteRecursionL() {
	var from, to int
l:
	goto l

	_, _ = from, to // want "unreachable"
}

func selectL() {
	var from, to int

	select {}

	_, _ = from, to // want "unreachable"
}
