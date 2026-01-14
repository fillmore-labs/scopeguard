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

package cantreturn

import (
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"k8s.io/klog"
	klog2 "k8s.io/klog/v2"
)

func zapLog() {
	log := zap.NewNop()

	log.Fatal("") // want "Can't return"
	log.Panic("") // want "Can't return"

	sugaredlog := log.Sugar()

	sugaredlog.Fatal()    // want "Can't return"
	sugaredlog.Fatalf("") // want "Can't return"
	sugaredlog.Fatalln()  // want "Can't return"
	sugaredlog.Fatalw("") // want "Can't return"
	sugaredlog.Panic()    // want "Can't return"
	sugaredlog.Panicf("") // want "Can't return"
	sugaredlog.Panicln()  // want "Can't return"
	sugaredlog.Panicw("") // want "Can't return"
}

func logrusLog() {
	log := logrus.New()

	log.Exit(1)    // want "Can't return"
	log.Panic()    // want "Can't return"
	log.Panicf("") // want "Can't return"
	log.Panicln()  // want "Can't return"

	entry := logrus.NewEntry(log)

	entry.Panic()    // want "Can't return"
	entry.Panicf("") // want "Can't return"
	entry.Panicln()  // want "Can't return"
}

func kLog() {
	klog.Exit()        // want "Can't return"
	klog.ExitDepth(0)  // want "Can't return"
	klog.Exitf("")     // want "Can't return"
	klog.Exitln()      // want "Can't return"
	klog.Fatal()       // want "Can't return"
	klog.FatalDepth(0) // want "Can't return"
	klog.Fatalf("")    // want "Can't return"
	klog.Fatalln()     // want "Can't return"

	klog2.Exit()        // want "Can't return"
	klog2.ExitDepth(0)  // want "Can't return"
	klog2.Exitf("")     // want "Can't return"
	klog2.Exitln()      // want "Can't return"
	klog2.Fatal()       // want "Can't return"
	klog2.FatalDepth(0) // want "Can't return"
	klog2.Fatalf("")    // want "Can't return"
	klog2.Fatalln()     // want "Can't return"
}
