// SPDX-License-Identifier: AGPL-3.0-or-later

package versioncmd

import "testing"

func TestVersionString(t *testing.T) {
	BuildVersion = "0.1.0"
	if got := String(); got != "kendali 0.1.0" {
		t.Fatalf("got %q", got)
	}
}
