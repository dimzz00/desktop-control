// SPDX-License-Identifier: AGPL-3.0-or-later

// Package rustdesk wraps the RustDesk CLI so kendali can detect, configure, and
// install the (unmodified) upstream binary.
package rustdesk

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"strings"

	"github.com/dimzz00/kendali/internal/platform"
)

// Status describes what kendali observed about the RustDesk installation.
type Status struct {
	// Installed reports whether the RustDesk binary was found.
	Installed bool
	// Version is the version string reported by `rustdesk --version`.
	Version string
	// ServiceActive reports whether the RustDesk service is running.
	ServiceActive bool
}

// Manager drives RustDesk through a platform.Runner so it can be tested with
// fakes.
type Manager struct {
	Run  platform.Runner
	GOOS string
}

func (m Manager) binary() string {
	if m.GOOS == "windows" {
		return "rustdesk.exe"
	}
	return "rustdesk"
}

func (m Manager) isMissing(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, exec.ErrNotFound) {
		return true
	}
	if errors.Is(err, fs.ErrNotExist) {
		return true
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		// Some runners surface a non-zero exit; treat 127 as "not found".
		if ee.ExitCode() == 127 {
			return true
		}
	}
	return false
}

// Detect reports whether RustDesk is installed and whether its service is
// active.
func (m Manager) Detect(ctx context.Context) (Status, error) {
	out, _, err := m.Run.Run(ctx, m.binary(), "--version")
	if m.isMissing(err) {
		return Status{}, nil
	}
	if err != nil {
		return Status{}, fmt.Errorf("rustdesk: detect: %w", err)
	}
	st := Status{Installed: true, Version: parseVersion(string(out))}
	st.ServiceActive = m.probeServiceActive(ctx)
	return st, nil
}

// ID returns the RustDesk device id.
func (m Manager) ID(ctx context.Context) (string, error) {
	out, _, err := m.Run.Run(ctx, m.binary(), "--get-id")
	if err != nil {
		return "", fmt.Errorf("rustdesk: get id: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// SupportsPasswordFlag reports whether the installed RustDesk accepts the
// --password flag.
func (m Manager) SupportsPasswordFlag(ctx context.Context) (bool, error) {
	out, errOut, err := m.Run.Run(ctx, m.binary(), "--help")
	combined := string(out) + "\n" + string(errOut)
	if err != nil && !strings.Contains(combined, "--password") {
		// Help may exit non-zero; that is fine as long as we saw the flag.
		if m.isMissing(err) {
			return false, fmt.Errorf("rustdesk: binary missing: %w", err)
		}
	}
	return strings.Contains(combined, "--password"), nil
}

// ErrPasswordUnsupported is returned when the installed RustDesk does not accept
// the --password flag, so the password must be set manually in the UI.
var ErrPasswordUnsupported = errors.New("rustdesk: --password flag not supported")

// SetPassword sets the RustDesk permanent password when the flag is supported.
func (m Manager) SetPassword(ctx context.Context, pw string) error {
	ok, err := m.SupportsPasswordFlag(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: set the password manually in the RustDesk UI (Settings -> Security)", ErrPasswordUnsupported)
	}
	_, _, err = m.Run.Run(ctx, m.binary(), "--password", pw)
	if err != nil {
		return fmt.Errorf("rustdesk: set password: %w", err)
	}
	return nil
}

// EnableDirectIP enables direct-IP connections. When the CLI flag is not
// supported it returns an error describing the manual UI steps (spec spike
// 14.1).
func (m Manager) EnableDirectIP(ctx context.Context) error {
	_, _, err := m.Run.Run(ctx, m.binary(), "--option", "direct-server", "Y")
	if err != nil {
		return fmt.Errorf("rustdesk: could not enable direct-server via CLI (%w); "+
			"enable it manually in the RustDesk UI (Settings -> Network -> Direct IP access) "+
			"and re-run `rc install`", err)
	}
	return nil
}

// InstallPackage installs a RustDesk package artifact.
func (m Manager) InstallPackage(ctx context.Context, artifact string) error {
	if m.GOOS == "windows" {
		_, _, err := m.Run.Run(ctx, "msiexec", "/i", artifact, "/qn", "/norestart")
		if err != nil {
			return fmt.Errorf("rustdesk: msiexec: %w", err)
		}
		return nil
	}
	_, _, err := m.Run.Run(ctx, "dpkg", "-i", artifact)
	if err != nil {
		return fmt.Errorf("rustdesk: dpkg: %w", err)
	}
	return nil
}
