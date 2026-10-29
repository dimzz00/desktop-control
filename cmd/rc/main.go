// SPDX-License-Identifier: AGPL-3.0-or-later

// Command rc is the kendali remote-control CLI.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/dimzz00/kendali/internal/platform"
	"github.com/dimzz00/kendali/internal/rustdesk"
	"github.com/dimzz00/kendali/internal/versioncmd"
)

const usage = `usage: rc <command> [flags]

commands:
  version     print the kendali version
  status      print connection status
  id          print the RustDesk device id
  doctor      diagnose the installation
  logs        show RustDesk logs
  token       print or rotate the terminal token
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
	case "status":
		return runStatus(ctx, args[1:], stdout, stderr)
	case "id":
		runner := platform.Exec{}
		id, err := rustdesk.Manager{Run: runner, GOOS: runtime.GOOS}.ID(ctx)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "rc: id: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintln(stdout, id)
		return 0
	case "doctor":
		return runDoctor(ctx, args[1:], stdout, stderr)
	case "logs":
		return runLogs(ctx, args[1:], stdout, stderr)
	case "token":
		return runToken(ctx, args[1:], stdout, stderr)
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
