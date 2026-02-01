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

func loop2L(from, to int) {
	for i := 0; i < 10; i++ {
	}

	_ = to // want "is reachable"
}

func infiniteL(from, to int) int {
	_ = from
	for {
	}

	return to // want "unreachable"
}

func gotoBackL(from, to int) {
	_ = from
L:
	goto L

	_ = to // want "unreachable"
}

func forever(from int) (to int) {
	for {
		continue

		break
	}

	return to // want "unreachable"
}

func forever2(from int) (to int) {
	for i := 0; ; i++ {
	}

	return to // want "unreachable"
}
