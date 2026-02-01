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

import (
	"log"
	"os"
	"runtime"
	"syscall"
)

func logFatal() {
	log.Fatal() // want "Can't return"
}

func builtinPanic() {
	panic("") // want "Can't return"
}

func logFatalf() {
	l := log.Default()

	l.Fatalf("") // want "Can't return"
}

func osExit() {
	os.Exit(1) // want "Can't return"
}

func syscallExit() {
	syscall.Exit(1) // want "Can't return"
}

func runtimeGoexit() {
	runtime.Goexit() // want "Can't return"
}

func normalReturn() {
	println("hello") // OK
}

func funcReturn() {
	panic := log.Fatal

	panic("hello") // OK
}
