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

package analyzer

import (
	"flag"

	"fillmore-labs.com/scopeguard/internal/analyze"
)

// RegisterFlags binds the [Options] values to command line flag values.
// A nil flag set value defaults to the program's command line.
func registerFlags(o *analyze.Options, flags *flag.FlagSet) {
	if flags == nil {
		flags = flag.CommandLine
	}

	flags.BoolVar(&o.Generated, "generated", o.Generated, "check generated files")
	flags.IntVar(&o.MaxLines, "max-lines", o.MaxLines, "maximum number of extra lines to suggest a move")
}
