// SPDX-License-Identifier: AGPL-3.0-or-later

// Command kendali-term is the terminal sidecar for kendali.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/dimzz00/kendali/internal/terminal"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "version" {
		_, _ = fmt.Fprintln(stdout, terminal.VersionString())
		return 0
	}
	_, _ = fmt.Fprintln(stdout, terminal.VersionString())
	return 0
}
