// SPDX-License-Identifier: AGPL-3.0-or-later

package rustdesk

import (
	"context"
	"strings"
)

// probeVersion asks RustDesk for its version string. It is split out so tests
// can exercise the parsing without a full Detect call.
func (m Manager) probeVersion(ctx context.Context) (string, error) {
	out, _, err := m.Run.Run(ctx, m.binary(), "--version")
	if err != nil {
		return "", err
	}
	return parseVersion(string(out)), nil
}

// probeServiceActive checks whether the RustDesk service is currently running.
func (m Manager) probeServiceActive(ctx context.Context) bool {
	if m.GOOS == "windows" {
		out, _, err := m.Run.Run(ctx, "sc", "query", "RustDesk")
		return err == nil && strings.Contains(strings.ToUpper(string(out)), "RUNNING")
	}
	out, _, err := m.Run.Run(ctx, "systemctl", "is-active", "rustdesk")
	return err == nil && strings.TrimSpace(string(out)) == "active"
}

// parseVersion extracts a version token from `rustdesk --version` output.
func parseVersion(s string) string {
	for _, line := range strings.Split(s, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.EqualFold(fields[0], "rustdesk") {
			return fields[1]
		}
	}
	for _, line := range strings.Split(s, "\n") {
		for _, f := range strings.Fields(line) {
			if len(f) > 0 && f[0] >= '0' && f[0] <= '9' {
				return f
			}
		}
	}
	return strings.TrimSpace(s)
}
