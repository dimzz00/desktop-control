// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"runtime"

	"github.com/dimzz00/kendali/internal/config"
	"github.com/dimzz00/kendali/internal/platform"
)

func runToken(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("token", flag.ContinueOnError)
	fs.SetOutput(stderr)
	rotate := fs.Bool("rotate", false, "rotate the terminal token")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	layout := platform.ForOS(runtime.GOOS)
	cfg, err := config.Load(layout.ConfigFile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "rc: token: %v\n", err)
		return 1
	}
	if *rotate {
		tok, err := config.NewToken()
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "rc: token: %v\n", err)
			return 1
		}
		cfg.Terminal.Token = tok
		if err := config.Save(layout.ConfigFile, cfg); err != nil {
			_, _ = fmt.Fprintf(stderr, "rc: token: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintln(stdout, tok)
		return 0
	}
	_, _ = fmt.Fprintln(stdout, cfg.Terminal.Token)
	return 0
}
