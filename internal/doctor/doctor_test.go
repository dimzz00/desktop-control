// SPDX-License-Identifier: AGPL-3.0-or-later

package doctor

import (
	"context"
	"net"
	"testing"

	"github.com/dimzz00/kendali/internal/config"
	"github.com/dimzz00/kendali/internal/netinfo"
	"github.com/dimzz00/kendali/internal/rustdesk"
	"github.com/dimzz00/kendali/internal/tailnet"
)

func TestEvaluateFlagsVersionDrift(t *testing.T) {
	fs := Evaluate(context.Background(), Deps{RS: rustdesk.Status{Installed: true, Version: "1.4.3"}, PinnedRustDesk: "1.4.2"})
	var found bool
	for _, f := range fs {
		if f.ID == "rustdesk-version-drift" {
			found = true
			if f.Severity != SeverityWarn {
				t.Fatalf("severity = %s", f.Severity)
			}
			if !contains(f.Message, "1.4.3") || !contains(f.Message, "1.4.2") {
				t.Fatalf("message = %s", f.Message)
			}
		}
	}
	if !found {
		t.Fatal("drift finding missing")
	}
}

func TestEvaluateFailsWhenNoAddresses(t *testing.T) {
	fs := Evaluate(context.Background(), Deps{})
	var found bool
	for _, f := range fs {
		if f.ID == "addresses" {
			found = true
			if f.Severity != SeverityFail {
				t.Fatalf("severity = %s", f.Severity)
			}
		}
	}
	if !found || ExitOK(fs) {
		t.Fatal("expected fail + ExitOK false")
	}
}

func TestEvaluateTailscaleLoggedOut(t *testing.T) {
	fs := Evaluate(context.Background(), Deps{Tail: tailnet.State{Installed: true, Running: false}})
	var found bool
	for _, f := range fs {
		if f.ID == "tailscale" {
			found = true
			if f.Severity != SeverityWarn {
				t.Fatalf("severity = %s", f.Severity)
			}
			if f.Fix == "" || !contains(f.Fix, "tailscale up") {
				t.Fatalf("fix = %s", f.Fix)
			}
		}
	}
	if !found {
		t.Fatal("tailscale finding missing")
	}
}

func TestEvaluateAllGood(t *testing.T) {
	cfg := config.Default()
	fs := Evaluate(context.Background(), Deps{
		Config:         cfg,
		RS:             rustdesk.Status{Installed: true, Version: "1.4.2", ServiceActive: true},
		Tail:           tailnet.State{Installed: true, Running: true, IPv4: net.ParseIP("100.1.2.3")},
		Addrs:          []netinfo.Address{{Kind: netinfo.KindLAN}},
		PinnedRustDesk: "1.4.2",
		Token:          "tok",
	})
	if !ExitOK(fs) {
		t.Fatalf("not all ok: %+v", fs)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
