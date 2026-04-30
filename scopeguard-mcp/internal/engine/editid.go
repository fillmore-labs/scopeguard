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
	"hash/fnv"
	"strconv"
)

// editKey identifies a diagnostic by its source location and message,
// used as input to [editKey.editID]. It is not stored or transmitted.
type editKey struct {
	Filename string // absolute path to the source file
	Message  string // diagnostic message text
	Offset   int    // byte offset within the file
}

// editID computes a fingerprint from file path, byte offset, and message.
func (k editKey) editID() EditID {
	var buf [6]byte

	h := fnv.New32a()
	_, _ = h.Write([]byte(k.Filename))                             // ignore error
	_, _ = h.Write([]byte{':'})                                    // ignore error
	_, _ = h.Write(strconv.AppendInt(buf[:], int64(k.Offset), 10)) // ignore error
	_, _ = h.Write([]byte{':'})                                    // ignore error
	_, _ = h.Write([]byte(k.Message))                              // ignore error

	e := h.Sum32()
	if e == 0 {
		e = 1
	}

	return EditID(e)
}

// EditID is the unique ID of an edit.
type EditID uint32

// String implements [fmt.Stringer].
func (e EditID) String() string {
	s, _ := e.MarshalText() // ignore error
	return string(s)
}

// UnmarshalText implements [encoding.TextUnmarshaler].
func (e *EditID) UnmarshalText(text []byte) error {
	v, err := strconv.ParseUint(string(text), 16, 32)
	if err != nil {
		return err
	}

	if v == 0 {
		return &strconv.NumError{Func: "ParseEditID", Num: string(text), Err: strconv.ErrRange}
	}

	*e = EditID(v)

	return nil
}

// MarshalText implements [encoding.TextMarshaler].
func (e EditID) MarshalText() ([]byte, error) {
	return e.AppendText(nil)
}

// AppendText implements [encoding.TextAppender].
func (e EditID) AppendText(buf []byte) ([]byte, error) {
	if e == 0 {
		return buf, nil
	}

	return fmt.Appendf(buf, "%08x", uint32(e)), nil
}
