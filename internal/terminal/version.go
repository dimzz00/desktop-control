// SPDX-License-Identifier: AGPL-3.0-or-later

// Package terminal provides the PTY, session, and auth primitives for
// kendali-term.
package terminal

// BuildVersion is injected via -ldflags at build time.
var BuildVersion = "dev"

// VersionString returns the kendali-term version string.
func VersionString() string {
	return "kendali-term " + BuildVersion
}
