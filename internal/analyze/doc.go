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

// Package analyze implements the scopeguard static analysis pass.
//
// # Overview
//
// ScopeGuard detects Go variables that can be moved to tighter scopes.
//
// # Example
//
// Before:
//
//	func process(data []byte) error {
//	    err := validate(data)  // err lives in entire function scope
//	    if err != nil {
//	        return err
//	    }
//	    // ... rest of function
//	}
//
// After applying scopeguard's suggested fix:
//
//	func process(data []byte) error {
//	    if err := validate(data); err != nil {  // err lives only in if scope
//	        return err
//	    }
//	    // ... rest of function
//	}
//
// # Architecture
//
// The analyzer uses a three-stage pipeline:
//
//  1. Usage: Collect variable declarations (both `:=` and `var` statements) and compute minimum usage scope
//  2. Target: Select target AST nodes and resolve conflicts (e.g., Init field conflicts)
//  3. Report: Generate diagnostics with suggested fixes
//
// # Supported Target Scopes
//
// The analyzer can move declarations to:
//
//   - Init fields: if, for, switch, type switch statements
//   - Block scopes: BlockStmt, CaseClause (switch/select cases)
//
// # Safety Constraints
//
// The analyzer prevents moves that would change program semantics:
//
//   - Loop bodies: Variables can move TO a for loop's Init field, but NOT into
//     the loop body (which would change the variable's lifetime)
//   - Function literals: Variables CANNOT cross function boundaries (would
//     change closure capture semantics)
//
// # Current Limitations
//
//   - Does not combine multiple single-variable declarations into one multi-value
//     declaration (a := 1; b := 2 -> a, b := 1, 2)
//
// # Special handling:
//
//   - Composite literals in Init fields require parentheses (Go spec requirement)
//   - While-style for loops need an extra semicolon when adding an Init field
package analyze
