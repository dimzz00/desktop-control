// SPDX-License-Identifier: AGPL-3.0-or-later

package terminal

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestPTYEcho(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	p, err := NewPTY()
	if err != nil {
		t.Fatal(err)
	}
	shell := "/bin/sh"
	if _, err := os.Stat(shell); err != nil {
		t.Skip("no /bin/sh")
	}
	if err := p.Start(shell, []string{"-c", "echo hello"}); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 256)
	deadline := time.Now().Add(2 * time.Second)
	var out string
	for time.Now().Before(deadline) {
		n, err := p.Read(buf)
		if n > 0 {
			out += string(buf[:n])
		}
		if err != nil {
			break
		}
		if strings.Contains(out, "hello") {
			break
		}
	}
	_ = p.Close()
	if !strings.Contains(out, "hello") {
		t.Fatalf("output = %q, want hello", out)
	}
}
