// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"github.com/dimzz00/kendali/internal/config"
	"github.com/dimzz00/kendali/internal/deps"
	"github.com/dimzz00/kendali/internal/fetch"
	"github.com/dimzz00/kendali/internal/install"
	"github.com/dimzz00/kendali/internal/platform"
	"github.com/dimzz00/kendali/internal/rustdesk"
)

func runInstall(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", "", "path to rc.yaml")
	adopt := fs.Bool("adopt-existing", false, "adopt a pre-existing RustDesk install")
	archive := fs.String("archive", "", "local tarball/zip to install from")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	layout := platform.ForOS(runtime.GOOS)
	// Override config path default.
	if *configPath == "" {
		*configPath = layout.ConfigFile
	}

	// Read deps.lock next to the binary? Default to ./deps.lock then /etc.
	depsPath := "deps.lock"
	if _, err := os.Stat(depsPath); err != nil {
		depsPath = "/etc/kendali/deps.lock"
	}
	entries, err := deps.Load(depsPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "rc: install: load deps: %v\n", err)
		return 1
	}

	// Load config (optional).
	var cfg *config.Config
	if c, err := config.Load(*configPath); err == nil {
		cfg = c
	}

	runner := platform.Exec{}
	env := install.Env{
		OS:         runtime.GOOS,
		Run:        runner,
		Layout:     layout,
		Config:     cfg,
		ConfigPath: *configPath,
		Deps:       entries,
		Fetch: func(ctx context.Context, url, sha, dest string) (string, error) {
			return fetch.Download(ctx, http.DefaultClient, url, sha, dest)
		},
		RS: rustdesk.Manager{Run: runner, GOOS: runtime.GOOS},
	}

	opts := install.Options{AdoptExisting: *adopt, ArtifactPath: *archive}
	res, err := install.Run(ctx, env, opts)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "rc: install: %v\n", err)
		return 1
	}
	for _, a := range res.Actions {
		_, _ = fmt.Fprintln(stdout, a)
	}
	if res.Changed {
		_, _ = fmt.Fprintln(stdout, "install complete")
	} else {
		_, _ = fmt.Fprintln(stdout, "already up to date")
	}
	return 0
}
