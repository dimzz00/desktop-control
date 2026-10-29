// SPDX-License-Identifier: AGPL-3.0-or-later

package terminal

import "testing"

func TestVersionString(t *testing.T) {
	old := BuildVersion
	BuildVersion = "0.2.0"
	t.Cleanup(func() { BuildVersion = old })
	if got := VersionString(); got != "kendali-term 0.2.0" {
		t.Fatalf("got %q", got)
	}
}
