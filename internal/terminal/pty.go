// SPDX-License-Identifier: AGPL-3.0-or-later

package terminal

import "io"

// PTY hosts a shell session.
type PTY interface {
	io.ReadWriteCloser
	Start(shell string, args []string) error
	Resize(cols, rows int) error
}

// NewPTY returns a platform-appropriate PTY implementation.
func NewPTY() (PTY, error) {
	return newPTY()
}
