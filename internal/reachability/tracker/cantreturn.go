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

package tracker

import (
	"go/ast"
	"go/types"
)

// _knownFuncs are functions that do not return.
var _knownFuncs = map[FuncName]struct{}{
	{Path: "log", Name: "Fatal"}:   {},
	{Path: "log", Name: "Fatalf"}:  {},
	{Path: "log", Name: "Panic"}:   {},
	{Path: "log", Name: "Panicf"}:  {},
	{Path: "log", Name: "Panicln"}: {},

	{Path: "log", Receiver: "Logger", Name: "Fatal"}:   {},
	{Path: "log", Receiver: "Logger", Name: "Fatalf"}:  {},
	{Path: "log", Receiver: "Logger", Name: "Panic"}:   {},
	{Path: "log", Receiver: "Logger", Name: "Panicf"}:  {},
	{Path: "log", Receiver: "Logger", Name: "Panicln"}: {},

	{Path: "os", Name: "Exit"}:        {},
	{Path: "syscall", Name: "Exit"}:   {},
	{Path: "runtime", Name: "Goexit"}: {},

	{Path: "testing", Receiver: "common", Name: "Fatal"}:   {},
	{Path: "testing", Receiver: "common", Name: "Fatalf"}:  {},
	{Path: "testing", Receiver: "common", Name: "FailNow"}: {},
	{Path: "testing", Receiver: "common", Name: "Skip"}:    {},
	{Path: "testing", Receiver: "common", Name: "Skipf"}:   {},
	{Path: "testing", Receiver: "common", Name: "SkipNow"}: {},

	{Path: "testing", Receiver: "TB", Name: "Fatal"}:   {},
	{Path: "testing", Receiver: "TB", Name: "Fatalf"}:  {},
	{Path: "testing", Receiver: "TB", Name: "FailNow"}: {},
	{Path: "testing", Receiver: "TB", Name: "Skip"}:    {},
	{Path: "testing", Receiver: "TB", Name: "Skipf"}:   {},
	{Path: "testing", Receiver: "TB", Name: "SkipNow"}: {},

	{Path: "github.com/sirupsen/logrus", Receiver: "Entry", Name: "Panic"}:    {},
	{Path: "github.com/sirupsen/logrus", Receiver: "Entry", Name: "Panicf"}:   {},
	{Path: "github.com/sirupsen/logrus", Receiver: "Entry", Name: "Panicln"}:  {},
	{Path: "github.com/sirupsen/logrus", Receiver: "Logger", Name: "Exit"}:    {},
	{Path: "github.com/sirupsen/logrus", Receiver: "Logger", Name: "Panic"}:   {},
	{Path: "github.com/sirupsen/logrus", Receiver: "Logger", Name: "Panicf"}:  {},
	{Path: "github.com/sirupsen/logrus", Receiver: "Logger", Name: "Panicln"}: {},
	{Path: "go.uber.org/zap", Receiver: "Logger", Name: "Fatal"}:              {},
	{Path: "go.uber.org/zap", Receiver: "Logger", Name: "Panic"}:              {},
	{Path: "go.uber.org/zap", Receiver: "SugaredLogger", Name: "Fatal"}:       {},
	{Path: "go.uber.org/zap", Receiver: "SugaredLogger", Name: "Fatalf"}:      {},
	{Path: "go.uber.org/zap", Receiver: "SugaredLogger", Name: "Fatalln"}:     {},
	{Path: "go.uber.org/zap", Receiver: "SugaredLogger", Name: "Fatalw"}:      {},
	{Path: "go.uber.org/zap", Receiver: "SugaredLogger", Name: "Panic"}:       {},
	{Path: "go.uber.org/zap", Receiver: "SugaredLogger", Name: "Panicf"}:      {},
	{Path: "go.uber.org/zap", Receiver: "SugaredLogger", Name: "Panicln"}:     {},
	{Path: "go.uber.org/zap", Receiver: "SugaredLogger", Name: "Panicw"}:      {},
	{Path: "k8s.io/klog", Name: "Exit"}:                                       {},
	{Path: "k8s.io/klog", Name: "ExitDepth"}:                                  {},
	{Path: "k8s.io/klog", Name: "Exitf"}:                                      {},
	{Path: "k8s.io/klog", Name: "Exitln"}:                                     {},
	{Path: "k8s.io/klog", Name: "Fatal"}:                                      {},
	{Path: "k8s.io/klog", Name: "FatalDepth"}:                                 {},
	{Path: "k8s.io/klog", Name: "Fatalf"}:                                     {},
	{Path: "k8s.io/klog", Name: "Fatalln"}:                                    {},
	{Path: "k8s.io/klog/v2", Name: "Exit"}:                                    {},
	{Path: "k8s.io/klog/v2", Name: "ExitDepth"}:                               {},
	{Path: "k8s.io/klog/v2", Name: "Exitf"}:                                   {},
	{Path: "k8s.io/klog/v2", Name: "Exitln"}:                                  {},
	{Path: "k8s.io/klog/v2", Name: "Fatal"}:                                   {},
	{Path: "k8s.io/klog/v2", Name: "FatalDepth"}:                              {},
	{Path: "k8s.io/klog/v2", Name: "Fatalf"}:                                  {},
	{Path: "k8s.io/klog/v2", Name: "Fatalln"}:                                 {},
}

// CantReturn iteratively unwraps an expression to find the underlying function declaration.
func CantReturn(info *types.Info, n *ast.CallExpr) bool {
	ex := n.Fun

unwrap:
	switch e := ex.(type) {
	case *ast.Ident:
		return cantReturnFunc(info, e)

	case *ast.SelectorExpr:
		return cantReturnFunc(info, e.Sel)

	case *ast.IndexExpr: // Generic function instantiation with a type parameter ("myFunc[T]").
		ex = e.X // Unwrap to the function identifier.
		goto unwrap

	case *ast.IndexListExpr: // Generic function instantiation with multiple type parameters ("myFunc[T, U]").
		ex = e.X // Unwrap to the function identifier.
		goto unwrap

	case *ast.ParenExpr: // Parenthesized expression ("(myFunc)")
		ex = e.X // Unwrap to the inner expression.
		goto unwrap

	default: // Pointer dereference or another function reference.
		return false
	}
}

func cantReturnFunc(info *types.Info, id *ast.Ident) bool {
	use := info.Uses[id]
	if fun, ok := use.(*types.Func); ok {
		name := FuncNameOf(fun)
		_, ok := _knownFuncs[name]

		return ok
	}

	return use == builtinPanic
}

var builtinPanic = types.Universe.Lookup("panic").(*types.Builtin)
