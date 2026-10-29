// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"flag"
	"io"
	"runtime"

	"github.com/dimzz00/kendali/internal/config"
	"github.com/dimzz00/kendali/internal/doctor"
	"github.com/dimzz00/kendali/internal/netinfo"
	"github.com/dimzz00/kendali/internal/platform"
	"github.com/dimzz00/kendali/internal/rustdesk"
	"github.com/dimzz00/kendali/internal/tailnet"
)

func runDoctor(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	layout := platform.ForOS(runtime.GOOS)
	cfg, _ := config.Load(layout.ConfigFile)
	runner := platform.Exec{}
	rs, _ := rustdesk.Manager{Run: runner, GOOS: runtime.GOOS}.Detect(ctx)
	tail, _ := tailnet.Detect(ctx, runner)
	addrs, _ := netinfo.Detect()
	token := ""
	if cfg != nil {
		token = cfg.Terminal.Token
	}
	pinned := ""
	if cfg != nil {
		pinned = cfg.Device.RustDeskVersion
	}
	deps := doctor.Deps{
		OS:             runtime.GOOS,
		Config:         cfg,
		ConfigPath:     layout.ConfigFile,
		RS:             rs,
		Tail:           tail,
		Addrs:          addrs,
		PinnedRustDesk: pinned,
		Token:          token,
	}
	findings := doctor.Evaluate(ctx, deps)
	doctor.WriteText(stdout, findings)
	if !doctor.ExitOK(findings) {
		return 1
	}
	return 0
}
