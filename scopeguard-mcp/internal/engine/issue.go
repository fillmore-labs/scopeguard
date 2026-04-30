// Copyright 2025-2026 Oliver Eikemeier. All Rights Reserved.
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

// Package category defines the diagnostic classification types used throughout
// the ScopeGuard analyzer pipeline.
package engine

import (
	"fmt"
	"strings"
)

// Issue is a diagnostic classification type used to identify specific categories of issues.
type Issue uint8

const (
	// IssueUnknown represents an unrecognized diagnostic category.
	IssueUnknown Issue = iota

	// IssueScope represents issues related to scope.
	IssueScope

	// IssueShadow represents shadowed variable issues.
	IssueShadow

	// IssueNestedAssignment represents nested reassignment issues, typically in closures requiring restructuring.
	IssueNestedAssignment
)

// safetyNames lists the recognized issue names.
var issueNames = [...]string{
	"unknown",
	"scope",
	"shadow",
	"nested",
}

func (i Issue) String() string {
	if int(i) >= len(issueNames) {
		return fmt.Sprintf("Issue(%d)", i)
	}

	return issueNames[i]
}

// ValidIssueNames returns a slice of valid issue names, excluding "unknown".
func ValidIssueNames() []string {
	return issueNames[1:]
}

// MarshalText implements [encoding.TextMarshaler].
func (i Issue) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

// UnmarshalText implements [encoding.TextUnmarshaler].
func (i *Issue) UnmarshalText(text []byte) error {
	v := string(text)
	for j, t := range issueNames[1:] {
		if v == t {
			*i = Issue(j + 1)
			return nil
		}
	}

	return fmt.Errorf("unknown issue: %q (valid values: %s)", v, strings.Join(ValidIssueNames(), ", "))
}
