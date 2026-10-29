// SPDX-License-Identifier: AGPL-3.0-or-later

package terminal

import (
	"context"
	"sync"
	"time"
)

// Session represents one terminal connection.
type Session struct {
	ID           string
	PTY          PTY
	LastActivity time.Time
	mu           sync.Mutex
}

// Touch updates LastActivity.
func (s *Session) Touch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastActivity = time.Now()
}

// Idle reports whether the session has been idle longer than d.
func (s *Session) Idle(d time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return time.Since(s.LastActivity) > d
}

// Manager tracks sessions and reaps idle ones.
type Manager struct {
	Idle     time.Duration
	Sessions map[string]*Session
	mu       sync.Mutex
}

func NewManager(idle time.Duration) *Manager {
	return &Manager{Idle: idle, Sessions: map[string]*Session{}}
}

func (m *Manager) Create(ctx context.Context, p PTY) (*Session, error) {
	s := &Session{ID: newID(), PTY: p, LastActivity: time.Now()}
	m.mu.Lock()
	m.Sessions[s.ID] = s
	m.mu.Unlock()
	return s, nil
}

func (m *Manager) Touch(id string) {
	m.mu.Lock()
	s, ok := m.Sessions[id]
	m.mu.Unlock()
	if ok {
		s.Touch()
	}
}

func (m *Manager) Close(id string) {
	m.mu.Lock()
	s, ok := m.Sessions[id]
	delete(m.Sessions, id)
	m.mu.Unlock()
	if ok && s.PTY != nil {
		_ = s.PTY.Close()
	}
}

func (m *Manager) Reap() {
	m.mu.Lock()
	var stale []string
	for id, s := range m.Sessions {
		if s.Idle(m.Idle) {
			stale = append(stale, id)
		}
	}
	m.mu.Unlock()
	for _, id := range stale {
		m.Close(id)
	}
}

func newID() string {
	return time.Now().UTC().Format("20060102T150405.000000000")
}
