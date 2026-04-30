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

package typeutil

import (
	"bytes"
	"errors"
	"fmt"
	"go/types"
	"slices"
	"strings"
)

// FuncName represents the fully qualified name of a function or method:
// Package path, receiver type, and function name, ignoring type parameters.
type FuncName struct {
	// Path is the package path (e.g. "encoding/json").
	Path string

	// LocalFuncName is the package-local part (receiver and name).
	LocalFuncName
}

// LocalFuncName is a package-local name of a function or method.
type LocalFuncName struct {
	// Receiver is the name of the receiver type (e.g. "Decoder").
	// It is empty for regular functions.
	Receiver string

	// Name is the function or method name (e.g. "Decode").
	Name string
}

// FuncNameOf extracts the name components of a given *[types.Func].
// It populates a FuncName struct, which is simplified and canonicalized
// from fun.Fullname() and can then be used as a map index or to get a
// string representation.
func FuncNameOf(fun *types.Func) (name FuncName) {
	name.Name = fun.Name()

	recv := fun.Signature().Recv()
	if recv == nil {
		// It's a regular function.
		if pkg := fun.Pkg(); pkg != nil {
			name.Path = pkg.Path()
		}

		return name
	}

	// It's a method.
	rtyp := recv.Type()

unwarp:
	switch t := rtyp.(type) {
	case *types.Alias:
		rtyp = t.Rhs() // Unwrap alias.
		goto unwarp

	case *types.Pointer:
		rtyp = t.Elem() // If it's a pointer, unwrap to the element type.
		goto unwarp

	case *types.Named:
		tn := t.Obj()
		if pkg := tn.Pkg(); pkg != nil {
			name.Path = pkg.Path()
		}
		name.Receiver = tn.Name()

	case *types.Interface: // Method on an interface type.
		name.Receiver = "interface"

	default: // Anonymous types shouldn't have methods.
		name.Receiver = "<invalid>"
	}

	return name
}

// String returns the fully qualified function name as a string.
//
// The string representation uses parentheses around the receiver to avoid ambiguity:
// `(encoding/json.Decoder).Decode` for methods vs. `encoding/json.Unmarshal` for functions.
func (f FuncName) String() string {
	buf, _ := f.MarshalText()
	return string(buf)
}

// String returns the local function name as a string.
//
// Since there is no package path, there is no ambiguity: the format is simply
// "Receiver.Name" for methods and "Name" for functions.
func (l LocalFuncName) String() string {
	buf, _ := l.MarshalText()
	return string(buf)
}

// Compare compares two [FuncName] instances lexicographically.
// It first compares by Path, and if they are equal, it compares by Receiver/Name.
func (f FuncName) Compare(other FuncName) int {
	if c := strings.Compare(f.Path, other.Path); c != 0 {
		return c
	}

	return f.LocalFuncName.Compare(other.LocalFuncName)
}

// Compare compares two [LocalFuncName] instances lexicographically by Receiver, then Name.
func (l LocalFuncName) Compare(other LocalFuncName) int {
	if c := strings.Compare(l.Receiver, other.Receiver); c != 0 {
		return c
	}

	return strings.Compare(l.Name, other.Name)
}

// MarshalText implements [encoding.TextMarshaler].
func (f FuncName) MarshalText() ([]byte, error) {
	return f.AppendText(nil)
}

// AppendText implements [encoding.TextAppender].
func (f FuncName) AppendText(buf []byte) ([]byte, error) {
	size := len(f.Name)

	plen := len(f.Path)
	if plen > 0 {
		size += plen + 1
	}

	rlen := len(f.Receiver)
	if rlen > 0 {
		size += rlen + 3
	}

	buf = slices.Grow(buf, size)

	if rlen > 0 {
		buf = append(buf, '(')
	}

	if plen > 0 {
		buf = append(buf, f.Path...)
		buf = append(buf, '.')
	}

	if rlen > 0 {
		buf = append(buf, f.Receiver...)
		buf = append(buf, ')', '.')
	}

	buf = append(buf, f.Name...)

	return buf, nil
}

// MarshalText implements [encoding.TextMarshaler].
func (l LocalFuncName) MarshalText() ([]byte, error) {
	return l.AppendText(nil)
}

// AppendText implements [encoding.TextAppender].
func (l LocalFuncName) AppendText(buf []byte) ([]byte, error) {
	size := len(l.Name)

	rlen := len(l.Receiver)
	if rlen > 0 {
		size += rlen + 1
	}

	buf = slices.Grow(buf, size)

	if rlen > 0 {
		buf = append(buf, l.Receiver...)
		buf = append(buf, '.')
	}

	buf = append(buf, l.Name...)

	return buf, nil
}

var errInvalidFuncName = errors.New("invalid function name")

// UnmarshalText implements [encoding.TextUnmarshaler].
func (f *FuncName) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return fmt.Errorf("%w %q", errInvalidFuncName, text)
	}

	var path, receiver, name []byte

	if text[0] == '(' {
		// It's a method
		closeParen := bytes.IndexByte(text, ')')
		if closeParen < 0 || closeParen+1 >= len(text) || text[closeParen+1] != '.' {
			return fmt.Errorf("%w %q", errInvalidFuncName, text)
		}

		name = text[closeParen+2:]

		receiverPart := text[1:closeParen]
		// Ignore leading pointer type
		if len(receiverPart) > 0 && receiverPart[0] == '*' {
			receiverPart = receiverPart[1:]
		}

		if lastDot := bytes.LastIndexByte(receiverPart, '.'); lastDot < 0 {
			receiver = receiverPart
		} else {
			path = receiverPart[:lastDot]
			receiver = receiverPart[lastDot+1:]
		}

		if len(receiver) == 0 {
			return fmt.Errorf("%w %q", errInvalidFuncName, text)
		}
	} else {
		// It's a regular function
		if lastDot := bytes.LastIndexByte(text, '.'); lastDot < 0 {
			name = text
		} else {
			path = text[:lastDot]
			name = text[lastDot+1:]
		}
	}

	if len(name) == 0 {
		return fmt.Errorf("%w %q", errInvalidFuncName, text)
	}

	*f = FuncName{
		Path: string(path),
		LocalFuncName: LocalFuncName{
			Receiver: string(receiver),
			Name:     string(name),
		},
	}

	return nil
}

// UnmarshalText implements [encoding.TextUnmarshaler].
func (l *LocalFuncName) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return fmt.Errorf("%w %q", errInvalidFuncName, text)
	}

	var receiver, name []byte

	if lastDot := bytes.LastIndexByte(text, '.'); lastDot < 0 {
		name = text
	} else {
		receiver = text[:lastDot]
		name = text[lastDot+1:]
	}

	if len(name) == 0 {
		return fmt.Errorf("%w %q", errInvalidFuncName, text)
	}

	*l = LocalFuncName{
		Receiver: string(receiver),
		Name:     string(name),
	}

	return nil
}
