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

import "strconv"

// EditKey identifies a diagnostic by its source location and message,
// used as input to [EditKey.editID]. It is not stored or transmitted.
type EditKey struct {
	Filename string // absolute path to the source file
	Message  string // diagnostic message text
	Offset   int    // byte offset within the file
}

// EditID computes a fingerprint from file path, byte offset, and message.
func (k EditKey) EditID() EditID {
	var buf [20]byte
	offset := strconv.AppendInt(buf[:0], int64(k.Offset), 10)

	h := new32a()
	h.write([]byte(k.Filename))
	h.write([]byte{':'})
	h.write(offset)
	h.write([]byte{':'})
	h.write([]byte(k.Message))

	v := h.sum32()
	if v == 0 {
		v = 1
	}

	return EditID(v)
}

// EditID is the unique ID of an edit.
type EditID uint32

// String implements [fmt.Stringer].
func (e EditID) String() string {
	if e == 0 {
		return ""
	}

	var buf [8]byte
	s := appendHex(buf[:0], uint32(e))

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

	return appendHex(buf, uint32(e)), nil
}

type sum32a uint32

const (
	offset32 = 2166136261
	prime32  = 16777619
)

func new32a() sum32a {
	return offset32
}

func (s *sum32a) sum32() uint32 { return uint32(*s) }

func (s *sum32a) write(data []byte) {
	hash := *s
	for _, c := range data {
		hash ^= sum32a(c)
		hash *= prime32
	}
	*s = hash
}

func appendHex(buf []byte, v uint32) []byte {
	const hex = "0123456789abcdef"

	return append(buf,
		hex[v>>28],
		hex[(v>>24)&0xf],
		hex[(v>>20)&0xf],
		hex[(v>>16)&0xf],
		hex[(v>>12)&0xf],
		hex[(v>>8)&0xf],
		hex[(v>>4)&0xf],
		hex[v&0xf],
	)
}
