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

package bitmask

// Bitmask is a sample bitmask.
//
//go:generate go tool bitmask -type Bitmask -json -truevalue All
type Bitmask uint8

// Sample values for [Bitmask].
const (
	One   Bitmask = 1 << iota   // one
	Two                         // two
	Three                       // three
	All   Bitmask = 1<<iota - 1 // all
	None  Bitmask = 0           // zero
)
