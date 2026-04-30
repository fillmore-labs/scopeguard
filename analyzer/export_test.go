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

package analyzer

import "fillmore-labs.com/scopeguard/internal/typeutil"

// WithFunctions is an internal [Option] to restrict analysis to the named functions.
func WithFunctions(functions ...typeutil.LocalFuncName) Option { return withFunctions(functions...) }

// WithRenames is an internal [Option].
func WithRenames(renames typeutil.RenameMap) Option {
	return withRenames(renames)
}
