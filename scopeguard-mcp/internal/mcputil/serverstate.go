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

package mcputil

import (
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerState maintains the runtime state of the MCP server.
type ServerState struct {
	diffstore       map[*mcp.ServerSession]sessionStore
	mu              sync.RWMutex
	InlineResources bool
}

// sessionStore holds the diffs from the last tool call for a specific session.
type sessionStore struct {
	diffs     map[string]string
	updatedAt time.Time
}

// getStore retrieves the diffs stored from the last tool call for a given session.
func (s *ServerState) getStore(session *mcp.ServerSession) (sessionStore, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	store, ok := s.diffstore[session]

	return store, ok
}

// commitStore saves the diffs for a given session, evicting the oldest session if the store is full.
func (s *ServerState) commitStore(session *mcp.ServerSession, diffs map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.evictOldestIfFull(session)

	s.diffstore[session] = sessionStore{
		diffs:     diffs,
		updatedAt: time.Now(),
	}
}

// maxSessions limits the number of sessions we store state for to prevent memory leaks.
const maxSessions = 10

// evictOldestIfFull ensures there is space for a new session, evicting the oldest if the store is full.
// It must be called with [ServerState.mu] locked.
func (s *ServerState) evictOldestIfFull(session *mcp.ServerSession) {
	if s.diffstore == nil {
		s.diffstore = make(map[*mcp.ServerSession]sessionStore)
		return
	}

	// No need to evict if we're updating an existing session or if we have capacity.
	if _, ok := s.diffstore[session]; ok || len(s.diffstore) < maxSessions {
		return
	}

	// We have len(s.diffstore) >= maxSessions
	oldestSession := calcOldestSession(s.diffstore)
	delete(s.diffstore, oldestSession)
}

func calcOldestSession(diffstore map[*mcp.ServerSession]sessionStore) (oldestSession *mcp.ServerSession) {
	var oldestTime time.Time
	for sess, store := range diffstore {
		if oldestSession == nil || store.updatedAt.Before(oldestTime) {
			oldestSession, oldestTime = sess, store.updatedAt
		}
	}

	return oldestSession
}
