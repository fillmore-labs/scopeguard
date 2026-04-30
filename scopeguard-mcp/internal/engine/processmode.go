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

package engine

import (
	"fmt"
	"strings"
)

// ProcessMode controls how [Process] handles diagnostics.
type ProcessMode uint8

const (
	// ProcessPreview never writes files. Every diagnostic is diff-rendered.
	// This is the default (zero value).
	ProcessPreview ProcessMode = iota

	// ProcessApply applies only the edits whose IDs are listed in the 'apply' argument.
	// Unknown IDs are an error. All other edits are still diff-rendered.
	ProcessApply

	// ProcessApplySafe applies every safe fix in one pass. Unsafe and breaking
	// fixes are still diff-rendered, never written; to write those, the caller
	// must opt in via [ProcessApply] with explicit IDs.
	ProcessApplySafe
)

// ProcessModes represents the predefined set of process modes as string values.
var ProcessModes = [...]string{
	ProcessPreview:   "preview",
	ProcessApply:     "apply",
	ProcessApplySafe: "apply_safe",
}

// UnmarshalText implements [encoding.TextUnmarshaler].
func (p *ProcessMode) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*p = ProcessPreview
		return nil
	}

	s := string(text)

	for i, t := range ProcessModes {
		if s == t {
			*p = ProcessMode(i)

			return nil
		}
	}

	return fmt.Errorf("unknown scope mode: %q, valid modes: %s",
		s, strings.Join(ProcessModes[:], ", "))
}

// MarshalText implements [encoding.TextMarshaler].
func (p ProcessMode) MarshalText() ([]byte, error) {
	return p.AppendText(nil)
}

// AppendText implements [encoding.TextAppender].
func (p ProcessMode) AppendText(buf []byte) ([]byte, error) {
	if int(p) >= len(ProcessModes) {
		return nil, fmt.Errorf("unknown process mode %d", p)
	}

	buf = append(buf, ProcessModes[p]...)

	return buf, nil
}
