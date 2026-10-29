// SPDX-License-Identifier: AGPL-3.0-or-later

// Package tailnet detects the Tailscale state and its IPv4 address.
package tailnet

import (
	"context"
	"net"
	"strings"

	"github.com/dimzz00/kendali/internal/platform"
)

// State describes the observed Tailscale state.
type State struct {
	// Installed reports whether the tailscale binary was found.
	Installed bool
	// Running reports whether tailscale is connected and has a tailnet IPv4.
	Running bool
	// IPv4 is the tailnet address when Running is true.
	IPv4 net.IP
}

var tailnetRange = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

// Detect probes tailscale via the runner.
func Detect(ctx context.Context, run platform.Runner) (State, error) {
	_, _, err := run.Run(ctx, "tailscale", "version")
	if err != nil {
		// Distinguish missing binary from other failures: both -> not installed
		// is acceptable for our purposes; a broken install is reported as not
		// installed rather than crashing rc status.
		return State{}, nil
	}
	st := State{Installed: true}

	out, _, err := run.Run(ctx, "tailscale", "ip", "-4")
	if err != nil {
		// "not logged in" or daemon down: not running, no error.
		return st, nil
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		ip := net.ParseIP(line)
		if ip == nil {
			continue
		}
		v4 := ip.To4()
		if v4 == nil {
			continue
		}
		if !tailnetRange.Contains(v4) {
			continue
		}
		st.Running = true
		st.IPv4 = v4
		return st, nil
	}
	// Output present but no tailnet IP: treat as not running.
	return st, nil
}
