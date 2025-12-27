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

package nofix

import "fmt"

func recoveredReturn() {
	f := func() (int, bool) { return 1, true }

	// This function has a named result parameter, but the usage is not detected
	v := func() (r int) {
		defer func() { _ = recover() }()
		r, ok := f() //nolint:scopeguard this would be moved
		if ok {
			_ = r // use r
		}

		panic("recovered") // no return statement
	}()

	fmt.Println(v)
}
