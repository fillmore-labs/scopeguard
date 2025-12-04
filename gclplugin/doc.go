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

/*
Package gclplugin provides golangci-lint plugin integration for the [scopeguard] analyzer.

# Usage

1. Add a file `.custom-gcl.yaml` to your source with:

	---
	version: v2.7.0

	name: golangci-lint
	destination: .

	plugins:
	  - module: fillmore-labs.com/scopeguard
	    import: fillmore-labs.com/scopeguard/gclplugin
	    version: v0.0.1

2. Run `golangci-lint custom` from your project root.

This will create a custom `golangci-lint` executable in your project root.

3. Configure the linter in `.golangci.yaml`:

	---
	version: "2"
	linters:
	  default: none
	  enable:
	    - scopeguard
	  settings:
	    custom:
	      scopeguard:
	        type: module
	        description: "scopeguard helps tighten variable scopes."
	        original-url: "https://fillmore-labs.com/scopeguard"

4. Run the linter:

	./golangci-lint run .

[scopeguard]: https://github.com/fillmore-labs/scopeguard#scopeguard
*/
package gclplugin
