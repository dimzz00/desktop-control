// SPDX-License-Identifier: AGPL-3.0-or-later

package terminal

import (
	"context"
	"testing"
	"time"
)

func TestSessionReap(t *testing.T) {
	m := NewManager(1 * time.Millisecond)
	s, _ := m.Create(context.Background(), nilPTY{})
	s.LastActivity = time.Now().Add(-time.Second)
	m.Reap()
	if len(m.Sessions) != 0 {
		t.Fatalf("session not reaped")
	}
}

func TestSessionTouchPreventsReap(t *testing.T) {
	m := NewManager(time.Hour)
	s, _ := m.Create(context.Background(), nilPTY{})
	m.Touch(s.ID)
	m.Reap()
	if len(m.Sessions) != 1 {
		t.Fatalf("session reaped prematurely")
	}
	m.Close(s.ID)
}

// helpers
type nilPTY struct{}

func (nilPTY) Read(b []byte) (int, error)              { return 0, nil }
func (nilPTY) Write(b []byte) (int, error)             { return len(b), nil }
func (nilPTY) Close() error                            { return nil }
func (nilPTY) Start(shell string, args []string) error { return nil }
func (nilPTY) Resize(cols, rows int) error             { return nil }
