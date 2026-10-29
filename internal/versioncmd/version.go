// SPDX-License-Identifier: AGPL-3.0-or-later

// Package versioncmd reports the kendali build version.
package versioncmd

// BuildVersion is the application version. Release builds inject it with
// -ldflags "-X github.com/dimzz00/kendali/internal/versioncmd.BuildVersion=<VERSION>".
var BuildVersion = "dev"

// String returns the human-readable version line.
func String() string { return "kendali " + BuildVersion }
