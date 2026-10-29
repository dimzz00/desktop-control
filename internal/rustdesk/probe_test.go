// SPDX-License-Identifier: AGPL-3.0-or-later

package rustdesk

import (
	"context"
	"os"
	"testing"
)

func TestProbeVersion(t *testing.T) {
	f := &fakeRunner{out: map[string][]byte{
		"rustdesk --version": []byte("rustdesk 1.4.2\n"),
	}}
	v, err := Manager{Run: f, GOOS: "linux"}.probeVersion(context.Background())
	if err != nil || v != "1.4.2" {
		t.Fatalf("got %q %v", v, err)
	}
}

func TestProbeServiceActive(t *testing.T) {
	f := &fakeRunner{out: map[string][]byte{
		"systemctl is-active rustdesk": []byte("active\n"),
	}}
	if !(Manager{Run: f, GOOS: "linux"}.probeServiceActive(context.Background())) {
		t.Fatal("expected active")
	}
	f2 := &fakeRunner{out: map[string][]byte{}}
	mgr := Manager{Run: f2, GOOS: "linux"}
	if mgr.probeServiceActive(context.Background()) {
		t.Fatal("expected inactive")
	}
}

func TestParseVersionVariants(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"rustdesk 1.4.2\n", "1.4.2"},
		{"1.2.3\n", "1.2.3"},
		{"garbage", "garbage"},
	}
	for _, c := range cases {
		got := parseVersion(c.in)
		if got != c.want {
			t.Errorf("parseVersion(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestProbeVersionMissing(t *testing.T) {
	f := &fakeRunner{err: os.ErrNotExist}
	_, err := Manager{Run: f, GOOS: "linux"}.probeVersion(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
