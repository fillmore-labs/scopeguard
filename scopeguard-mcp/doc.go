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
Package scopeguard-mcp provides an MCP server for the [ScopeGuard] analyzer.

# Usage

1. Install the server:

	go install fillmore-labs.com/scopeguard/scopeguard-mcp@latest

2. Add it to your tool of choice:

[Claude Code]:

	claude mcp add --transport stdio --scope project ScopeGuard -- scopeguard-mcp

[Gemini]:

	gemini mcp add --transport stdio --scope project ScopeGuard scopeguard-mcp

or add

	{
		"mcpServers": {
			"ScopeGuard": {
				"command": "scopeguard-mcp"
			}
		}
	}

to .mcp.json, mcp_config.json, or your client's configuration.

[ScopeGuard]: https://github.com/fillmore-labs/scopeguard#scopeguard
[Claude Code]: https://code.claude.com/docs/en/mcp#option-3-add-a-local-stdio-server
[Gemini]: https://geminicli.com/docs/tools/mcp-server/#adding-an-stdio-server
*/
package main
