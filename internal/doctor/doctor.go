// SPDX-License-Identifier: AGPL-3.0-or-later

// Package doctor evaluates the health of a kendali installation.
package doctor

import (
	"context"
	"fmt"
	"io"

	"github.com/dimzz00/kendali/internal/config"
	"github.com/dimzz00/kendali/internal/netinfo"
	"github.com/dimzz00/kendali/internal/rustdesk"
	"github.com/dimzz00/kendali/internal/semver"
	"github.com/dimzz00/kendali/internal/tailnet"
)

// Severity of a finding.
type Severity string

const (
	SeverityOK   Severity = "ok"
	SeverityWarn Severity = "warn"
	SeverityFail Severity = "fail"
)

// Finding is one diagnostic result.
type Finding struct {
	ID       string
	Severity Severity
	Message  string
	Fix      string
}

// Deps is the input to Evaluate.
type Deps struct {
	OS             string
	Config         *config.Config
	ConfigPath     string
	RS             rustdesk.Status
	Tail           tailnet.State
	Addrs          []netinfo.Address
	PinnedRustDesk string
	Token          string
}

// Evaluate produces a list of findings. It performs no I/O.
func Evaluate(ctx context.Context, d Deps) []Finding {
	out := []Finding{}

	// config-valid
	if d.Config != nil {
		if err := d.Config.Validate(); err != nil {
			out = append(out, Finding{ID: "config-valid", Severity: SeverityFail, Message: err.Error(), Fix: "fix rc.yaml or restore from backup"})
		} else {
			out = append(out, Finding{ID: "config-valid", Severity: SeverityOK, Message: "config valid"})
		}
	} else {
		out = append(out, Finding{ID: "config-valid", Severity: SeverityFail, Message: "config not loaded", Fix: "run rc install"})
	}

	// rustdesk-installed
	if d.RS.Installed {
		out = append(out, Finding{ID: "rustdesk-installed", Severity: SeverityOK, Message: "rustdesk installed: " + d.RS.Version})
	} else {
		out = append(out, Finding{ID: "rustdesk-installed", Severity: SeverityFail, Message: "rustdesk not installed", Fix: "run rc install"})
	}

	// rustdesk-service
	if d.RS.ServiceActive {
		out = append(out, Finding{ID: "rustdesk-service", Severity: SeverityOK, Message: "rustdesk service active"})
	} else {
		out = append(out, Finding{ID: "rustdesk-service", Severity: SeverityWarn, Message: "rustdesk service not active", Fix: "systemctl start rustdesk (or start the service)"})
	}

	// rustdesk-version-drift
	if d.RS.Installed && d.PinnedRustDesk != "" {
		if d.RS.Version == d.PinnedRustDesk {
			out = append(out, Finding{ID: "rustdesk-version-drift", Severity: SeverityOK, Message: "rustdesk version matches pinned"})
		} else {
			// try semver for nicer message
			a, aerr := semver.Parse(d.RS.Version)
			b, berr := semver.Parse(d.PinnedRustDesk)
			var msg string
			if aerr == nil && berr == nil {
				msg = fmt.Sprintf("rustdesk %s (pinned %s), drift %d", a.String(), b.String(), a.Compare(b))
			} else {
				msg = fmt.Sprintf("rustdesk %s vs pinned %s (unparseable version)", d.RS.Version, d.PinnedRustDesk)
			}
			out = append(out, Finding{ID: "rustdesk-version-drift", Severity: SeverityWarn, Message: msg, Fix: "rc doctor reports drift only; rc upgrade does not touch RustDesk"})
		}
	} else {
		out = append(out, Finding{ID: "rustdesk-version-drift", Severity: SeverityOK, Message: "no drift check (not installed or no pin)"})
	}

	// tailscale
	if d.Tail.Installed {
		if d.Tail.Running {
			out = append(out, Finding{ID: "tailscale", Severity: SeverityOK, Message: "tailscale running: " + d.Tail.IPv4.String()})
		} else {
			out = append(out, Finding{ID: "tailscale", Severity: SeverityWarn, Message: "tailscale installed but not running", Fix: "tailscale up"})
		}
	} else {
		out = append(out, Finding{ID: "tailscale", Severity: SeverityWarn, Message: "tailscale not installed", Fix: "install tailscale"})
	}

	// addresses
	if len(d.Addrs) == 0 {
		out = append(out, Finding{ID: "addresses", Severity: SeverityFail, Message: "no usable addresses", Fix: "connect LAN/USB/Tailscale"})
	} else {
		out = append(out, Finding{ID: "addresses", Severity: SeverityOK, Message: fmt.Sprintf("%d addresses", len(d.Addrs))})
	}

	// token-present
	if d.Token != "" {
		out = append(out, Finding{ID: "token-present", Severity: SeverityOK, Message: "terminal token present"})
	} else {
		out = append(out, Finding{ID: "token-present", Severity: SeverityWarn, Message: "terminal token missing", Fix: "rc install or rc token --rotate"})
	}

	_ = ctx
	return out
}

// WriteText prints findings to w.
func WriteText(w io.Writer, fs []Finding) {
	for _, f := range fs {
		fix := ""
		if f.Fix != "" {
			fix = "  (fix: " + f.Fix + ")"
		}
		_, _ = fmt.Fprintf(w, "[%s] %s: %s%s\n", f.Severity, f.ID, f.Message, fix)
	}
}

// ExitOK reports whether there are no fail-severity findings.
func ExitOK(fs []Finding) bool {
	for _, f := range fs {
		if f.Severity == SeverityFail {
			return false
		}
	}
	return true
}
