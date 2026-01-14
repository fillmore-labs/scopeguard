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

package nofix

func nested() {
	var err error

	{
		err := error(nil)
		_ = err
	}

	err = func() error {
		err = error(nil) // want "Nested reassignment of variable 'err'"
		return err
	}()

	_ = err
}

func nested2() {
	var err error

	{
		err := error(nil)
		_ = err
	}

	err, ok := func() (error, bool) {
		err = error(nil) // want "Nested reassignment of variable 'err'"

		return err, true
	}()

	_, _ = err, ok
}

func nestedOk() {
	var err error

	{
		err := error(nil)
		_ = err
	}

	err = func() error {
		err := error(nil)
		return err
	}()

	_ = func() error {
		err := error(nil)
		return err
	}

	_ = err
}
