// SPDX-License-Identifier: AGPL-3.0-or-later

package rustdesk

import (
	"context"
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
}
