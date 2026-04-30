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

package help

import (
	"embed"
	"fmt"
	"strings"
)

// Instructions for connected clients.
//
//go:embed _instructions.md
var Instructions string

// helpTopics is an [io/fs.FS] containing the help topics.
//
//go:embed [^_.]*.md
var helpTopics embed.FS

const helpSuffix = ".md"

// allTopics is the cached list of topic file names (without the .md suffix).
// Populated once at init from the embedded FS, which can never fail at runtime.
var allTopics = func() []string {
	entries, err := helpTopics.ReadDir(".")
	if err != nil {
		panic(fmt.Sprintf("help: read embedded topics: %v", err))
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "_") || !strings.HasSuffix(name, helpSuffix) {
			// Safety check, filtered by the regex of [helpTopics]
			panic(fmt.Sprintf("Invalid help topic %q", name))
		}

		names = append(names, strings.TrimSuffix(name, helpSuffix))
	}

	return names
}()

// AllTopics returns the topic list.
func AllTopics() []string {
	return allTopics
}

// ReadTopic returns the Markdown for a single topic, or all topics concatenated when topic is "".
func ReadTopic(topic string) (string, error) {
	if topic == "" {
		return readAllTopics()
	}

	body, err := helpTopics.ReadFile(topic + helpSuffix)
	if err != nil {
		return "", fmt.Errorf("unknown help topic %q; valid topics: %s", topic, strings.Join(AllTopics(), ", "))
	}

	return string(body), nil
}

// readAllTopics returns the Markdown for all help topics, separated by a blank line.
func readAllTopics() (string, error) {
	var b strings.Builder

	for i, name := range allTopics {
		if i > 0 {
			b.WriteString("\n\n")
		}

		body, err := helpTopics.ReadFile(name + helpSuffix)
		if err != nil {
			return "", err
		}

		b.Write(body)
	}

	return b.String(), nil
}
