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

package tools

// defaultLimit is the maximum number of issues/edits returned when no explicit limit is set.
const defaultLimit = 50

// resolveLimit determines the limit of items to process, returning a default value if the provided limit is nil.
func resolveLimit(limit *int) int {
	if limit == nil {
		return defaultLimit
	}

	return *limit
}

// maxLines specifies the maximum number of lines a declaration can span to be considered for moving.
const maxLines = 10
