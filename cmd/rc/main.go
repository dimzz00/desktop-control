// SPDX-License-Identifier: AGPL-3.0-or-later

// Command rc is the kendali remote-control CLI.
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/dimzz00/kendali/internal/versioncmd"
)

const usage = `usage: rc <command> [flags]

commands:
  version     print the kendali version
  install     install kendali (RustDesk + rc)
  uninstall   remove kendali
`

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

// run executes one CLI invocation and returns its process exit code.
// It is separated from main so it can be tested without exiting the process.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(stderr, usage)
		return 2
	}

	switch args[0] {
	case "version":
		_, _ = fmt.Fprintln(stdout, versioncmd.String())
		return 0
	case "install":
		return runInstall(ctx, args[1:], stdout, stderr)
	case "uninstall":
		return runUninstall(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "rc: unknown command %q\n\n", args[0])
		_, _ = fmt.Fprint(stderr, usage)
		return 2
	}
}
