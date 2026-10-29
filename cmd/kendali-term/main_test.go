// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"testing"

	"github.com/dimzz00/kendali/internal/terminal"
)

func TestRunVersion(t *testing.T) {
	old := terminal.BuildVersion
	terminal.BuildVersion = "0.2.0"
	t.Cleanup(func() { terminal.BuildVersion = old })
	var stdout, stderr bytes.Buffer
	if code := run([]string{"version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if stdout.String() != "kendali-term 0.2.0\n" {
		t.Fatalf("got %q", stdout.String())
	}
}
