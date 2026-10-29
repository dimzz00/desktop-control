// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"net"
	"testing"

	"github.com/dimzz00/kendali/internal/config"
	"github.com/dimzz00/kendali/internal/netinfo"
	"github.com/dimzz00/kendali/internal/rustdesk"
	"github.com/dimzz00/kendali/internal/tailnet"
)

func TestRenderStatusShowsAddresses(t *testing.T) {
	var b bytes.Buffer
	err := renderStatus(&b, StatusInput{
		AppVersion: "0.1.0",
		Config:     config.Default(),
		Addrs:      []netinfo.Address{{Kind: netinfo.KindLAN, IP: net.ParseIP("192.168.1.42"), Iface: "wlp2s0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{"kendali 0.1.0", "LAN", "192.168.1.42", "belum aktif"} {
		if !bytes.Contains([]byte(out), []byte(want)) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderStatusSaysNotConnected(t *testing.T) {
	var b bytes.Buffer
	err := renderStatus(&b, StatusInput{AppVersion: "0.1.0", Config: config.Default()})
	if err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !bytes.Contains([]byte(out), []byte("tidak terhubung")) {
		t.Fatalf("output missing 'tidak terhubung':\n%s", out)
	}
}

func TestRenderStatusTailscaleLoggedOut(t *testing.T) {
	var b bytes.Buffer
	err := renderStatus(&b, StatusInput{
		AppVersion: "0.1.0",
		Config:     config.Default(),
		Tail:       tailnet.State{Installed: true, Running: false},
		RS:         rustdesk.Status{Installed: true, Version: "1.4.2", ServiceActive: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !bytes.Contains([]byte(out), []byte("terpasang, belum login")) {
		t.Fatalf("output missing tailscale logged-out message:\n%s", out)
	}
}
