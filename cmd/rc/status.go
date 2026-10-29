// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/dimzz00/kendali/internal/config"
	"github.com/dimzz00/kendali/internal/netinfo"
	"github.com/dimzz00/kendali/internal/platform"
	"github.com/dimzz00/kendali/internal/rustdesk"
	"github.com/dimzz00/kendali/internal/tailnet"
	"github.com/dimzz00/kendali/internal/versioncmd"
)

// StatusInput is the data renderStatus needs.
type StatusInput struct {
	AppVersion string
	Config     *config.Config
	RS         rustdesk.Status
	Tail       tailnet.State
	Addrs      []netinfo.Address
}

// renderStatus writes a human-readable status block.
func renderStatus(w io.Writer, in StatusInput) error {
	alias := "kendali"
	if in.Config != nil && in.Config.Device.Alias != "" {
		alias = in.Config.Device.Alias
	}
	_, _ = fmt.Fprintf(w, "kendali %s  (%s)\n", in.AppVersion, alias)

	// RustDesk
	if in.RS.Installed {
		svc := "stopped"
		if in.RS.ServiceActive {
			svc = "active"
		}
		_, _ = fmt.Fprintf(w, "rustdesk: %s  (service %s)\n", in.RS.Version, svc)
	} else {
		_, _ = fmt.Fprintln(w, "rustdesk: not installed")
	}

	// Tailscale
	if in.Tail.Installed {
		if in.Tail.Running {
			_, _ = fmt.Fprintf(w, "tailscale: %s\n", in.Tail.IPv4.String())
		} else {
			_, _ = fmt.Fprintln(w, "tailscale: terpasang, belum login")
		}
	} else {
		_, _ = fmt.Fprintln(w, "tailscale: tidak terpasang")
	}

	// Addresses
	_, _ = fmt.Fprintln(w, "addresses:")
	if len(in.Addrs) == 0 {
		_, _ = fmt.Fprintln(w, "  (tidak terhubung)")
	} else {
		for _, a := range in.Addrs {
			_, _ = fmt.Fprintf(w, "  %-8s %s  (%s)\n", strings.ToUpper(string(a.Kind)), a.IP.String(), a.Iface)
		}
	}

	_, _ = fmt.Fprintln(w, "terminal:  belum aktif (0.2.0)")
	return nil
}

// runStatus assembles StatusInput from the real system and renders it.
func runStatus(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	_ = args
	cfg, _ := config.Load(platform.ForOS(runtime.GOOS).ConfigFile)
	if cfg == nil {
		cfg = config.Default()
	}
	runner := platform.Exec{}
	rs, _ := rustdesk.Manager{Run: runner, GOOS: runtime.GOOS}.Detect(ctx)
	tail, _ := tailnet.Detect(ctx, runner)
	addrs, _ := netinfo.Detect()
	in := StatusInput{
		AppVersion: versioncmd.BuildVersion,
		Config:     cfg,
		RS:         rs,
		Tail:       tail,
		Addrs:      addrs,
	}
	if err := renderStatus(stdout, in); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
