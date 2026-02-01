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

package cantreturn

import "testing"

func TestCantReturn(t *testing.T) {
	t.SkipNow() // want "Can't return"

	t.FailNow() // want "Can't return"

	t.Fail()

	helperCantReturn(t)
}

func BenchmarkCantReturn(b *testing.B) {
	b.Skip("...") // want "Can't return"

	b.Fatal("...") // want "Can't return"

	b.Log("...")

	helperCantReturn(b)
}

func FuzzCantReturn(f *testing.F) {
	f.Skipf("%s", "...") // want "Can't return"

	f.Fatalf("%s", "...") // want "Can't return"

	f.Logf("...")

	helperCantReturn(f)
}

func helperCantReturn(tb testing.TB) {
	tb.Helper()

	tb.SkipNow() // want "Can't return"

	tb.FailNow() // want "Can't return"
}
