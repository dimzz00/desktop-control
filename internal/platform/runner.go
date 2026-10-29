// SPDX-License-Identifier: AGPL-3.0-or-later

// Package platform abstracts the operating system: process execution, the
// install layout, and elevation checks. Every other package talks to the OS
// through the Runner interface so it can be tested with fakes.
package platform

import (
	"bytes"
	"context"
	"os/exec"
)

// Runner runs an external command and returns its captured stdout and stderr.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (stdout, stderr []byte, err error)
}

// Exec is the real implementation of Runner: it invokes os/exec.
type Exec struct{}

// Run implements Runner using os/exec.CommandContext. stdout and stderr are
// captured separately; a non-zero exit is reported as an *exec.ExitError so
// callers can use errors.As.
func (Exec) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	return outBuf.Bytes(), errBuf.Bytes(), err
}
