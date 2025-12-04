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

// Maximum lines that can be moved to if statement's Init field.
func maxLines() {
	x1 := func() int { // want "Variable 'x1' can be moved to tighter if scope"
		i := 1

		return i
	}()
	{
		if x1 > 0 {
			fmt.Println(x1)
		}
	}

	x2 := func() int { // want "Variable 'x2' can be moved to tighter block scope"
		i := 1
		i++

		return i
	}()
	{
		if x2 > 0 {
			fmt.Println(x2)
		}
	}
}
