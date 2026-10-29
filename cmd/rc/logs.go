// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

func runLogs(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if runtime.GOOS == "windows" {
		dir := os.Getenv("ProgramData")
		if dir == "" {
			dir = `C:\ProgramData`
		}
		_, _ = fmt.Fprintf(stderr, "rc: logs: windows log dir %s\\RustDesk\\log (not implemented)\n", dir)
		return 0
	}
	cmd := exec.CommandContext(ctx, "journalctl", "-u", "rustdesk", "-n", "200", "--no-pager")
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		_, _ = fmt.Fprintf(stderr, "rc: logs: %v\n", err)
		return 1
	}
	return 0
}
