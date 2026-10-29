// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build windows

package terminal

import (
	"io"

	"github.com/UserExistsError/conpty"
)

type windowsPTY struct {
	cp *conpty.ConPty
}

func newPTY() (PTY, error) {
	return &windowsPTY{}, nil
}

func (p *windowsPTY) Start(shell string, args []string) error {
	cmdline := shell
	for _, a := range args {
		cmdline += " " + a
	}
	cp, err := conpty.Start(cmdline)
	if err != nil {
		return err
	}
	p.cp = cp
	return nil
}

func (p *windowsPTY) Read(b []byte) (int, error)  { return p.cp.Read(b) }
func (p *windowsPTY) Write(b []byte) (int, error) { return p.cp.Write(b) }
func (p *windowsPTY) Close() error {
	if p.cp != nil {
		return p.cp.Close()
	}
	return nil
}
func (p *windowsPTY) Resize(cols, rows int) error { return nil }

var _ io.ReadWriteCloser = (*windowsPTY)(nil)
