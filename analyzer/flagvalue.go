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

package analyzer

import "strconv"

type boolValue[F any, B boolFlag[F]] struct {
	flags B
	value F
}

type boolFlag[F any] interface {
	comparable
	Set(flag F, value bool)
	Enabled(flag F) bool
}

// Set implements [flag.Value].
func (f boolValue[_, B]) Set(s string) error {
	b, err := parseBool(s)
	if err != nil {
		return err
	}

	f.flags.Set(f.value, b)

	return nil
}

// String implements [flag.Value].
func (f boolValue[_, B]) String() string {
	var null B
	if f.flags == null {
		return "false"
	}

	return strconv.FormatBool(f.flags.Enabled(f.value))
}

// Get implements [flag.Getter].
func (f boolValue[_, B]) Get() any {
	var null B
	if f.flags == null {
		return false
	}

	return f.flags.Enabled(f.value)
}

// IsBoolFlag returns true to indicate that this is a boolean [flag.Value].
func (f boolValue[_, _]) IsBoolFlag() bool { return true }

// parseBool returns the boolean value represented by the string.
func parseBool(str string) (bool, error) {
	switch str {
	case "1", "t", "T", "true", "TRUE", "True", "on", "On", "full", "Full":
		return true, nil
	case "0", "f", "F", "false", "FALSE", "False", "off", "Off":
		return false, nil
	}

	return false, &strconv.NumError{Func: "ParseBool", Num: str, Err: strconv.ErrSyntax}
}
