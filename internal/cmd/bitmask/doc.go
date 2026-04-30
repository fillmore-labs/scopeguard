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

/*
Command bitmask generates String, Set, Has, MarshalText, AppendText and UnmarshalText methods for
bit-flag types whose constants carry stringer-style linecomment names. With "-json" it additionally
emits MarshalJSON/UnmarshalJSON; the marshaler renders a JSON array of bit names, and the unmarshaler
accepts either a bool (true sets predefined bits, false clears all), a string (parsed via UnmarshalText),
or an array of names.

A companion file (<output>_jsonv2.go) is also written, guarded by "//go:build goexperiment.jsonv2". It
holds the [encoding/json/v2] MarshalJSONTo/UnmarshalJSONFrom methods, only when the [jsonv2] experiment is on.

Types may also be declared in test files. A type defined in an in-package
_test.go file or in an external "_test" package has its helpers written to a
_test.go file (for example a_bitmask_test.go and a_bitmask_jsonv2_test.go), in
the matching package, so the generated code shares its source's test build
constraint. A type name that is declared in both the package and its external
"_test" package is ambiguous and rejected.

Constants of a bitmask type must satisfy the following constraints:
  - Single-bit constants must start at bit position 0 (value 1 << iota) and be contiguous with no gaps.
  - Multi-bit combination constants (aliases) may also carry line comments.
    Uncommented constants are ignored and treated as internal helpers.
  - All values must be non-negative integers.

Usage:

	bitmask -type T[,U ...] [-boolflag] [-json] [-truevalue [TYPE.]IDENT] [-output FILE] [dir]

Intended to be run via go generate, like this:

	//go:generate go tool bitmask -type Bitmask
	type Bitmask int

	const (
		One Bitmask = 1 << iota   // one
		Two                       // two
		All Bitmask = 1<<iota - 1 // all
	)

This requires the bitmask command to be registered as a tool in go.mod:

	tool fillmore-labs.com/scopeguard/internal/cmd/bitmask

[jsonv2]: https://go.dev/blog/jsonv2-exp
*/
package main
