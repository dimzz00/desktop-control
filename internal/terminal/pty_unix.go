// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build unix

package terminal

import (
	"os"
	"os/exec"

	"github.com/creack/pty"
)

type unixPTY struct {
	f *os.File
	c *exec.Cmd
}

func newPTY() (PTY, error) {
	return &unixPTY{}, nil
}

// Start launches shell in the PTY. (Interface extension below.)
func (p *unixPTY) Start(shell string, args []string) error {
	c := exec.Command(shell, args...)
	f, err := pty.Start(c)
	if err != nil {
		return err
	}
	p.f = f
	p.c = c
	return nil
}

func (p *unixPTY) Read(b []byte) (int, error)  { return p.f.Read(b) }
func (p *unixPTY) Write(b []byte) (int, error) { return p.f.Write(b) }
func (p *unixPTY) Close() error {
	if p.c != nil && p.c.Process != nil {
		_ = p.c.Process.Kill()
	}
	if p.f != nil {
		return p.f.Close()
	}
	return nil
}
func (p *unixPTY) Resize(cols, rows int) error {
	return pty.Setsize(p.f, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}
