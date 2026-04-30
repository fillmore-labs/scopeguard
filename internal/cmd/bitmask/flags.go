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

package main

import (
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"
)

type options struct {
	output     string
	typeNames  multiValues
	trueValues multiValues
	boolFlag   bool
	json       bool
}

func (o *options) registerFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.output, "output", "", "output `file` name (default <type>_bitmask.go)")
	fs.Var(&o.typeNames, "type", "comma-separated list of `type` names (required)")
	fs.Var(&o.trueValues, "truevalue", "`identifier` assigned for the literal \"true\" (qualify as Type.Const when multiple -type values are given)")
	fs.BoolVar(&o.boolFlag, "boolflag", false, "generate a boolean flag.Value helper")
	fs.BoolVar(&o.json, "json", false, "generate MarshalJSON and UnmarshalJSON")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: bitmask -type T[,U...] [-boolflag] [-json] [-truevalue [TYPE.]IDENT] [-output FILE] [dir]")
		fs.PrintDefaults()
	}
}

type multiValues []string

// String implements [flag.Value].
func (m multiValues) String() string {
	return strings.Join(m, ", ")
}

// Set implements [flag.Value].
func (m *multiValues) Set(s string) error {
	names := strings.Split(s, ",")
	for i := range names {
		names[i] = strings.TrimSpace(names[i])
	}

	names = slices.DeleteFunc(names, func(s string) bool { return s == "" })
	*m = append(*m, names...)

	return nil
}

// Get implements [flag.Getter].
func (m multiValues) Get() any {
	return []string(m)
}

// applyFlags overlays the command-line options onto every spec.
// JSON and BoolFlag apply uniformly to every spec in this invocation;
// TrueValue is per-type and assigned by applyTrueValues.
func applyFlags(specs []Spec, o options) error {
	for i := range specs {
		specs[i].JSON = o.json
		specs[i].BoolFlag = o.boolFlag
	}

	if err := applyTrueValues(specs, o.trueValues); err != nil {
		return err
	}

	return nil
}

// applyTrueValues assigns each -truevalue entry to its matching spec. An
// unqualified entry "Const" requires exactly one spec; a qualified entry
// "Type.Const" must match one of the requested types. Each entry must name a
// bit or alias constant on its target type.
func applyTrueValues(specs []Spec, entries []string) error {
	for _, entry := range entries {
		typeName, constName, qualified := strings.Cut(entry, ".")
		if !qualified {
			if len(specs) != 1 {
				return fmt.Errorf("%q: must be qualified as Type.Const with multiple -type values", entry)
			}

			typeName, constName = specs[0].TypeName, entry
		}

		idx := slices.IndexFunc(specs, func(s Spec) bool { return s.TypeName == typeName })
		if idx < 0 {
			return fmt.Errorf("%q: type %s not in types", entry, typeName)
		}

		spec := &specs[idx]
		if spec.TrueValue != "" {
			return fmt.Errorf("multiple entries for type %s", typeName)
		}

		cn, ok := findConst(spec, constName)
		if !ok {
			return fmt.Errorf("%q: %s has no constant %s", entry, typeName, constName)
		}

		// A zero-valued constant cannot be the true value: the generated code uses
		// 0 as the sentinel that disables bool input, so true and false would both
		// map to the zero value.
		if cn.value == 0 {
			return fmt.Errorf("%q: %s.%s is the zero value, which already denotes \"false\"", entry, typeName, constName)
		}

		spec.TrueValue = constName
	}

	return nil
}

// findConst returns the bit or alias constant named constName, reporting whether
// it was found.
func findConst(spec *Spec, constName string) (constInfo, bool) {
	matchesID := func(c constInfo) bool { return c.Const == constName }

	if i := slices.IndexFunc(spec.Bits, matchesID); i >= 0 {
		return spec.Bits[i], true
	}

	if i := slices.IndexFunc(spec.Aliases, matchesID); i >= 0 {
		return spec.Aliases[i], true
	}

	return constInfo{}, false
}
