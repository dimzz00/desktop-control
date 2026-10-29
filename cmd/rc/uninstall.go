// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/dimzz00/kendali/internal/platform"
)

func runUninstall(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	fs.SetOutput(stderr)
	keepConfig := fs.Bool("keep-config", false, "keep rc.yaml")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	layout := platform.ForOS(runtime.GOOS)
	// Remove installed release dirs.
	_ = os.RemoveAll(layout.Releases)
	_ = os.RemoveAll(layout.BinDir)
	// Remove config unless asked to keep.
	if !*keepConfig {
		_ = os.Remove(layout.ConfigFile)
	}
	_, _ = fmt.Fprintln(stdout, "uninstalled")
	return 0
}
