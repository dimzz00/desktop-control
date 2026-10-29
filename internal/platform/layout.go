// SPDX-License-Identifier: AGPL-3.0-or-later

package platform

import (
	"context"
	"os"
	"runtime"
	"strings"
)

// Layout describes where kendali stores its files on one operating system.
type Layout struct {
	// Root is the installation root directory.
	Root string
	// Releases holds one subdirectory per installed version.
	Releases string
	// ConfigFile is the path of rc.yaml.
	ConfigFile string
	// LogDir is where kendali and RustDesk write logs.
	LogDir string
	// BinDir is the directory of the currently active rc binary.
	BinDir string
}

// ForOS returns the Layout for the given GOOS value. For an unknown OS it
// returns the Linux layout, which keeps callers simple in tests.
func ForOS(goos string) Layout {
	switch goos {
	case "windows":
		base := os.Getenv("ProgramData")
		if base == "" {
			base = `C:\ProgramData`
		}
		root := base + `\kendali`
		return Layout{
			Root:       root,
			Releases:   root + `\releases`,
			ConfigFile: root + `\rc.yaml`,
			LogDir:     root + `\logs`,
			BinDir:     root + `\current`,
		}
	default:
		root := "/opt/kendali"
		return Layout{
			Root:       root,
			Releases:   root + "/releases",
			ConfigFile: "/etc/kendali/rc.yaml",
			LogDir:     "/var/log/kendali",
			BinDir:     root + "/current",
		}
	}
}

// IsElevated reports whether the current process has administrative privileges.
// The probe is chosen from runtime.GOOS: `id -u` on Linux/Unix, `net session`
// on Windows. The runner is injectable so callers can test without spawning
// processes.
func IsElevated(ctx context.Context, run Runner) (bool, error) {
	switch runtime.GOOS {
	case "windows":
		_, _, err := run.Run(ctx, "net", "session")
		if err != nil {
			return false, nil
		}
		return true, nil
	default:
		out, _, err := run.Run(ctx, "id", "-u")
		if err != nil {
			return false, err
		}
		return strings.TrimSpace(string(out)) == "0", nil
	}
}
