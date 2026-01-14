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

package check

import (
	"go/types"
	"strings"
)

// FuncName represents the fully qualified name of a function or method.
// It deconstructs a function's identity into its constituent parts: package path,
// receiver type, function name and ignores type parameters.
type FuncName struct {
	// Path is the package path ("encoding/json").
	Path string

	// Receiver is the name of the receiver type ("Decoder").
	// It is empty for regular functions.
	Receiver string

	// Name is the function or method name ("Decode").
	Name string

	// Ptr is true if the receiver is a pointer type.
	Ptr bool
}

// String returns the fully qualified function name as a string.
// For a method, the format is "(*<path>.<receiver>).<name>".
// For a function, the format is "<path>.<name>".
func (f FuncName) String() string {
	if f.Receiver == "" {
		// A regular function.
		if f.Path == "" {
			return f.Name
		}

		return f.Path + "." + f.Name
	}

	// A method.
	var sb strings.Builder

	sb.WriteByte('(')

	if f.Ptr {
		sb.WriteByte('*')
	}

	if f.Path != "" {
		sb.WriteString(f.Path)
		sb.WriteByte('.')
	}

	sb.WriteString(f.Receiver)
	sb.WriteByte(')')
	sb.WriteByte('.')
	sb.WriteString(f.Name)

	return sb.String()
}

// FuncNameOf extracts the name components of a given *[types.Func].
// It populates a FuncName struct, which is simplified and canonicalized
// from fun.Fullname() and can then be used as a map index or to get a
// string representation.
func FuncNameOf(fun *types.Func) FuncName {
	f := FuncName{
		Name: fun.Name(),
	}

	recv := fun.Signature().Recv()
	if recv == nil { // It's a regular function.
		if pkg := fun.Pkg(); pkg != nil {
			f.Path = pkg.Path()
		}

		return f
	}

	rtyp := types.Unalias(recv.Type()) // It's a method.

	// If it's a pointer, set the ptr flag and unwrap to the element type.
	if p, ok := rtyp.(*types.Pointer); ok {
		f.Ptr = true
		rtyp = types.Unalias(p.Elem())
	}

	switch t := rtyp.(type) {
	case *types.Named:
		tn := t.Obj()
		if pkg := tn.Pkg(); pkg != nil {
			f.Path = pkg.Path()
		}
		f.Receiver = tn.Name()

	case *types.Interface: // This case handles methods on an interface type.
		f.Receiver = "interface"

	default: // Anonymous types shouldn't have methods.
		f.Receiver = "<invalid>"
	}

	return f
}
