// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"fmt"
	"io"
	"runtime"

	"github.com/dimzz00/kendali/internal/config"
	"github.com/dimzz00/kendali/internal/platform"
)

func runTerm(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: rc term <status|url|token>")
		return 2
	}
	layout := platform.ForOS(runtime.GOOS)
	cfg, err := config.Load(layout.ConfigFile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "rc: term: %v\n", err)
		return 1
	}
	switch args[0] {
	case "status":
		_, _ = fmt.Fprintf(stdout, "bind: %s\nport: %d\nidle: %s\n", cfg.Terminal.Bind, cfg.Terminal.Port, cfg.Terminal.IdleTimeout)
		return 0
	case "url":
		host := "127.0.0.1"
		if cfg.Terminal.Bind == "tailnet" {
			// caller resolves tailnet IP; print placeholder
			host = "<tailnet-ip>"
		}
		_, _ = fmt.Fprintf(stdout, "http://%s:%d/?token=%s\n", host, cfg.Terminal.Port, cfg.Terminal.Token)
		return 0
	case "token":
		_, _ = fmt.Fprintln(stdout, cfg.Terminal.Token)
		return 0
	}
	_, _ = fmt.Fprintf(stderr, "rc: term: unknown subcommand %q\n", args[0])
	return 2
}
