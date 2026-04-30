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

	"fillmore-labs.com/scopeguard/internal/typeutil"
)

// _knownFuncs are functions that do not return.
var _knownFuncs = map[typeutil.FuncName]struct{}{
	{Path: "os", LocalFuncName: typeutil.LocalFuncName{Name: "Exit"}}:        {},
	{Path: "syscall", LocalFuncName: typeutil.LocalFuncName{Name: "Exit"}}:   {},
	{Path: "runtime", LocalFuncName: typeutil.LocalFuncName{Name: "Goexit"}}: {},

	{Path: "log", LocalFuncName: typeutil.LocalFuncName{Name: "Fatal"}}:   {},
	{Path: "log", LocalFuncName: typeutil.LocalFuncName{Name: "Fatalf"}}:  {},
	{Path: "log", LocalFuncName: typeutil.LocalFuncName{Name: "Fatalln"}}: {},
	{Path: "log", LocalFuncName: typeutil.LocalFuncName{Name: "Panic"}}:   {},
	{Path: "log", LocalFuncName: typeutil.LocalFuncName{Name: "Panicf"}}:  {},
	{Path: "log", LocalFuncName: typeutil.LocalFuncName{Name: "Panicln"}}: {},

	{Path: "log", LocalFuncName: typeutil.LocalFuncName{Receiver: "Logger", Name: "Fatal"}}:   {},
	{Path: "log", LocalFuncName: typeutil.LocalFuncName{Receiver: "Logger", Name: "Fatalf"}}:  {},
	{Path: "log", LocalFuncName: typeutil.LocalFuncName{Receiver: "Logger", Name: "Fatalln"}}: {},
	{Path: "log", LocalFuncName: typeutil.LocalFuncName{Receiver: "Logger", Name: "Panic"}}:   {},
	{Path: "log", LocalFuncName: typeutil.LocalFuncName{Receiver: "Logger", Name: "Panicf"}}:  {},
	{Path: "log", LocalFuncName: typeutil.LocalFuncName{Receiver: "Logger", Name: "Panicln"}}: {},

	{Path: "testing", LocalFuncName: typeutil.LocalFuncName{Receiver: "common", Name: "Fatal"}}:   {},
	{Path: "testing", LocalFuncName: typeutil.LocalFuncName{Receiver: "common", Name: "Fatalf"}}:  {},
	{Path: "testing", LocalFuncName: typeutil.LocalFuncName{Receiver: "common", Name: "FailNow"}}: {},
	{Path: "testing", LocalFuncName: typeutil.LocalFuncName{Receiver: "common", Name: "Skip"}}:    {},
	{Path: "testing", LocalFuncName: typeutil.LocalFuncName{Receiver: "common", Name: "Skipf"}}:   {},
	{Path: "testing", LocalFuncName: typeutil.LocalFuncName{Receiver: "common", Name: "SkipNow"}}: {},

	{Path: "testing", LocalFuncName: typeutil.LocalFuncName{Receiver: "TB", Name: "Fatal"}}:   {},
	{Path: "testing", LocalFuncName: typeutil.LocalFuncName{Receiver: "TB", Name: "Fatalf"}}:  {},
	{Path: "testing", LocalFuncName: typeutil.LocalFuncName{Receiver: "TB", Name: "FailNow"}}: {},
	{Path: "testing", LocalFuncName: typeutil.LocalFuncName{Receiver: "TB", Name: "Skip"}}:    {},
	{Path: "testing", LocalFuncName: typeutil.LocalFuncName{Receiver: "TB", Name: "Skipf"}}:   {},
	{Path: "testing", LocalFuncName: typeutil.LocalFuncName{Receiver: "TB", Name: "SkipNow"}}: {},

	{Path: "github.com/golang/glog", LocalFuncName: typeutil.LocalFuncName{Name: "Exit"}}:                            {},
	{Path: "github.com/golang/glog", LocalFuncName: typeutil.LocalFuncName{Name: "Exitf"}}:                           {},
	{Path: "github.com/golang/glog", LocalFuncName: typeutil.LocalFuncName{Name: "Exitln"}}:                          {},
	{Path: "github.com/golang/glog", LocalFuncName: typeutil.LocalFuncName{Name: "ExitContext"}}:                     {},
	{Path: "github.com/golang/glog", LocalFuncName: typeutil.LocalFuncName{Name: "ExitContextf"}}:                    {},
	{Path: "github.com/golang/glog", LocalFuncName: typeutil.LocalFuncName{Name: "ExitDepth"}}:                       {},
	{Path: "github.com/golang/glog", LocalFuncName: typeutil.LocalFuncName{Name: "ExitDepthf"}}:                      {},
	{Path: "github.com/golang/glog", LocalFuncName: typeutil.LocalFuncName{Name: "ExitContextDepth"}}:                {},
	{Path: "github.com/golang/glog", LocalFuncName: typeutil.LocalFuncName{Name: "ExitContextDepthf"}}:               {},
	{Path: "github.com/golang/glog", LocalFuncName: typeutil.LocalFuncName{Name: "Fatal"}}:                           {},
	{Path: "github.com/golang/glog", LocalFuncName: typeutil.LocalFuncName{Name: "Fatalf"}}:                          {},
	{Path: "github.com/golang/glog", LocalFuncName: typeutil.LocalFuncName{Name: "Fatalln"}}:                         {},
	{Path: "github.com/golang/glog", LocalFuncName: typeutil.LocalFuncName{Name: "FatalContext"}}:                    {},
	{Path: "github.com/golang/glog", LocalFuncName: typeutil.LocalFuncName{Name: "FatalContextf"}}:                   {},
	{Path: "github.com/golang/glog", LocalFuncName: typeutil.LocalFuncName{Name: "FatalDepth"}}:                      {},
	{Path: "github.com/golang/glog", LocalFuncName: typeutil.LocalFuncName{Name: "FatalDepthf"}}:                     {},
	{Path: "github.com/golang/glog", LocalFuncName: typeutil.LocalFuncName{Name: "FatalContextDepth"}}:               {},
	{Path: "github.com/golang/glog", LocalFuncName: typeutil.LocalFuncName{Name: "FatalContextDepthf"}}:              {},
	{Path: "github.com/sirupsen/logrus", LocalFuncName: typeutil.LocalFuncName{Receiver: "Entry", Name: "Panic"}}:    {},
	{Path: "github.com/sirupsen/logrus", LocalFuncName: typeutil.LocalFuncName{Receiver: "Entry", Name: "Panicf"}}:   {},
	{Path: "github.com/sirupsen/logrus", LocalFuncName: typeutil.LocalFuncName{Receiver: "Entry", Name: "Panicln"}}:  {},
	{Path: "github.com/sirupsen/logrus", LocalFuncName: typeutil.LocalFuncName{Receiver: "Logger", Name: "Exit"}}:    {},
	{Path: "github.com/sirupsen/logrus", LocalFuncName: typeutil.LocalFuncName{Receiver: "Logger", Name: "Panic"}}:   {},
	{Path: "github.com/sirupsen/logrus", LocalFuncName: typeutil.LocalFuncName{Receiver: "Logger", Name: "Panicf"}}:  {},
	{Path: "github.com/sirupsen/logrus", LocalFuncName: typeutil.LocalFuncName{Receiver: "Logger", Name: "Panicln"}}: {},
	{Path: "go.uber.org/zap", LocalFuncName: typeutil.LocalFuncName{Receiver: "Logger", Name: "Fatal"}}:              {},
	{Path: "go.uber.org/zap", LocalFuncName: typeutil.LocalFuncName{Receiver: "Logger", Name: "Panic"}}:              {},
	{Path: "go.uber.org/zap", LocalFuncName: typeutil.LocalFuncName{Receiver: "SugaredLogger", Name: "Fatal"}}:       {},
	{Path: "go.uber.org/zap", LocalFuncName: typeutil.LocalFuncName{Receiver: "SugaredLogger", Name: "Fatalf"}}:      {},
	{Path: "go.uber.org/zap", LocalFuncName: typeutil.LocalFuncName{Receiver: "SugaredLogger", Name: "Fatalln"}}:     {},
	{Path: "go.uber.org/zap", LocalFuncName: typeutil.LocalFuncName{Receiver: "SugaredLogger", Name: "Fatalw"}}:      {},
	{Path: "go.uber.org/zap", LocalFuncName: typeutil.LocalFuncName{Receiver: "SugaredLogger", Name: "Panic"}}:       {},
	{Path: "go.uber.org/zap", LocalFuncName: typeutil.LocalFuncName{Receiver: "SugaredLogger", Name: "Panicf"}}:      {},
	{Path: "go.uber.org/zap", LocalFuncName: typeutil.LocalFuncName{Receiver: "SugaredLogger", Name: "Panicln"}}:     {},
	{Path: "go.uber.org/zap", LocalFuncName: typeutil.LocalFuncName{Receiver: "SugaredLogger", Name: "Panicw"}}:      {},
	{Path: "k8s.io/klog", LocalFuncName: typeutil.LocalFuncName{Name: "Exit"}}:                                       {},
	{Path: "k8s.io/klog", LocalFuncName: typeutil.LocalFuncName{Name: "ExitDepth"}}:                                  {},
	{Path: "k8s.io/klog", LocalFuncName: typeutil.LocalFuncName{Name: "Exitf"}}:                                      {},
	{Path: "k8s.io/klog", LocalFuncName: typeutil.LocalFuncName{Name: "Exitln"}}:                                     {},
	{Path: "k8s.io/klog", LocalFuncName: typeutil.LocalFuncName{Name: "Fatal"}}:                                      {},
	{Path: "k8s.io/klog", LocalFuncName: typeutil.LocalFuncName{Name: "FatalDepth"}}:                                 {},
	{Path: "k8s.io/klog", LocalFuncName: typeutil.LocalFuncName{Name: "Fatalf"}}:                                     {},
	{Path: "k8s.io/klog", LocalFuncName: typeutil.LocalFuncName{Name: "Fatalln"}}:                                    {},
	{Path: "k8s.io/klog/v2", LocalFuncName: typeutil.LocalFuncName{Name: "Exit"}}:                                    {},
	{Path: "k8s.io/klog/v2", LocalFuncName: typeutil.LocalFuncName{Name: "ExitDepth"}}:                               {},
	{Path: "k8s.io/klog/v2", LocalFuncName: typeutil.LocalFuncName{Name: "Exitf"}}:                                   {},
	{Path: "k8s.io/klog/v2", LocalFuncName: typeutil.LocalFuncName{Name: "Exitln"}}:                                  {},
	{Path: "k8s.io/klog/v2", LocalFuncName: typeutil.LocalFuncName{Name: "Fatal"}}:                                   {},
	{Path: "k8s.io/klog/v2", LocalFuncName: typeutil.LocalFuncName{Name: "FatalDepth"}}:                              {},
	{Path: "k8s.io/klog/v2", LocalFuncName: typeutil.LocalFuncName{Name: "Fatalf"}}:                                  {},
	{Path: "k8s.io/klog/v2", LocalFuncName: typeutil.LocalFuncName{Name: "Fatalln"}}:                                 {},
}

// CanReturn iteratively unwraps an expression to find the underlying function declaration.
func CanReturn(info *types.Info, n *ast.CallExpr) bool {
	obj := typeutil.FuncOf(info, n)

	return obj == nil || canReturnFunc(obj)
}

func canReturnFunc(obj types.Object) bool {
	switch obj := obj.(type) {
	case *types.Func:
		name := typeutil.FuncNameOf(obj)
		_, found := _knownFuncs[name]

		return !found

	case *types.Builtin:
		return obj != builtinPanic // We could also check obj.Name() != "panic"

	default:
		return true
	}
}

var builtinPanic = types.Universe.Lookup("panic").(*types.Builtin)
