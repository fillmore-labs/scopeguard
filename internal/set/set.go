// Copyright 2024-2025 Oliver Eikemeier. All Rights Reserved.
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

package set

import (
	"cmp"
	"iter"
	"maps"
	"slices"
)

// Set is a collection of unique elements of type T.
type Set[T comparable] map[T]struct{}

// New returns a new set of type T containing elems.
func New[T comparable](elems ...T) Set[T] {
	s := make(Set[T], len(elems))
	for _, e := range elems {
		s.Add(e)
	}

	return s
}

// Add adds the element t to the set s.
// If t is already in s, it does nothing.
func (s Set[T]) Add(t T) {
	s[t] = struct{}{}
}

// Contains returns true if the element t is in the set s, false otherwise.
func (s Set[T]) Contains(t T) bool {
	_, ok := s[t]

	return ok
}

// All returns in iterator over the elements of s.
func (s Set[T]) All() iter.Seq[T] {
	return maps.Keys(s)
}

// Sorted returns the elements of s as a sorted list.
func Sorted[T cmp.Ordered](s Set[T]) []T {
	return slices.Sorted(s.All())
}

// AllSorted returns an iterator over the elements of s in sorted order.
func AllSorted[T cmp.Ordered](s Set[T]) iter.Seq[T] {
	return slices.Values(Sorted(s))
}
