// SPDX-License-Identifier: AGPL-3.0-or-later

// Package install orchestrates an idempotent kendali installation on top of
// an (unmodified) RustDesk binary.
package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dimzz00/kendali/internal/config"
	"github.com/dimzz00/kendali/internal/deps"
	"github.com/dimzz00/kendali/internal/platform"
	"github.com/dimzz00/kendali/internal/rustdesk"
)

// ErrExistingMismatch is returned when RustDesk is already installed at a
// version different from the one pinned in deps.lock and the caller did not
// ask to adopt it.
var ErrExistingMismatch = errors.New("install: existing RustDesk version differs from pinned version")

// Env bundles everything Run needs from the outside world.
type Env struct {
	OS         string
	Run        platform.Runner
	Layout     platform.Layout
	Config     *config.Config
	ConfigPath string
	Deps       []deps.Entry
	Fetch      func(ctx context.Context, url, sha256hex, destDir string) (string, error)
	RS         rustdesk.Manager
}

// Options tune a single Run invocation.
type Options struct {
	AdoptExisting bool
	ArtifactPath  string
}

// Result reports what Run did.
type Result struct {
	Changed bool
	Actions []string
}

func pinnedRustDesk(entries []deps.Entry) string {
	for _, e := range entries {
		if e.Name == "rustdesk" {
			return e.Version
		}
	}
	return ""
}

func rustdeskEntry(entries []deps.Entry) (deps.Entry, bool) {
	for _, e := range entries {
		if e.Name == "rustdesk" {
			return e, true
		}
	}
	return deps.Entry{}, false
}

func waitForService(ctx context.Context, m rustdesk.Manager) error {
	deadline := time.Now().Add(30 * time.Second)
	for {
		st, err := m.Detect(ctx)
		if err == nil && st.ServiceActive {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("install: rustdesk service did not become active")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// Run performs an idempotent installation. See package docs.
func Run(ctx context.Context, env Env, opts Options) (Result, error) {
	res := Result{Actions: []string{}}

	if env.ConfigPath == "" {
		env.ConfigPath = env.Layout.ConfigFile
	}
	if env.OS == "" {
		env.OS = "linux"
	}

	// Acquire an exclusive lock so two installs cannot race.
	if err := os.MkdirAll(env.Layout.Root, 0o755); err != nil {
		return res, fmt.Errorf("install: create root: %w", err)
	}
	lockPath := filepath.Join(env.Layout.Root, ".install.lock")
	lf, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return res, fmt.Errorf("install: acquire lock: %w", err)
	}
	defer func() {
		_ = lf.Close()
		_ = os.Remove(lockPath)
	}()

	// Detect existing RustDesk.
	st, err := env.RS.Detect(ctx)
	if err != nil {
		return res, fmt.Errorf("install: detect rustdesk: %w", err)
	}

	pinned := pinnedRustDesk(env.Deps)

	adopted := false
	if st.Installed && pinned != "" && st.Version != pinned {
		if !opts.AdoptExisting {
			return res, fmt.Errorf("%w: installed %s, pinned %s", ErrExistingMismatch, st.Version, pinned)
		}
		adopted = true
		pinned = st.Version
	}

	// Load existing config (if any).
	cfg := env.Config
	configExisted := cfg != nil
	if cfg == nil {
		c, err := config.Load(env.ConfigPath)
		if err == nil {
			cfg = c
			configExisted = true
		} else if !errors.Is(err, os.ErrNotExist) {
			return res, fmt.Errorf("install: load config: %w", err)
		}
	}
	if cfg == nil {
		cfg = config.Default()
	}

	// Already installed at the right version with a config: no-op.
	if st.Installed && (st.Version == pinned || adopted) && configExisted {
		env.Config = cfg
		return Result{Changed: false, Actions: res.Actions}, nil
	}

	changed := false

	// Install the package unless we are adopting an existing install or it is
	// already present at the pinned version.
	needInstall := !adopted && (!st.Installed || st.Version != pinned)
	// If adopted, never install. If already at pinned version, never install.
	if st.Installed && st.Version == pinned {
		needInstall = false
	}
	if adopted {
		needInstall = false
	}

	if needInstall {
		artifact := opts.ArtifactPath
		if artifact == "" {
			entry, ok := rustdeskEntry(env.Deps)
			if !ok {
				return res, errors.New("install: no rustdesk entry in deps.lock")
			}
			if env.Fetch == nil {
				return res, errors.New("install: fetch function not configured")
			}
			a, err := env.Fetch(ctx, entry.URL, entry.SHA256, env.Layout.Releases)
			if err != nil {
				return res, fmt.Errorf("install: fetch rustdesk: %w", err)
			}
			artifact = a
		}
		if err := env.RS.InstallPackage(ctx, artifact); err != nil {
			return res, err
		}
		if err := waitForService(ctx, env.RS); err != nil {
			return res, err
		}
		changed = true
		res.Actions = append(res.Actions, "installed rustdesk")
	}

	// Set password if missing.
	if cfg.Device.Password == "" {
		pw, err := config.NewToken()
		if err != nil {
			return res, fmt.Errorf("install: generate password: %w", err)
		}
		if err := env.RS.SetPassword(ctx, pw); err != nil {
			if errors.Is(err, rustdesk.ErrPasswordUnsupported) {
				res.Actions = append(res.Actions, "rustdesk password must be set manually (UI)")
			} else {
				return res, err
			}
		} else {
			cfg.Device.Password = pw
			changed = true
			res.Actions = append(res.Actions, "set rustdesk password")
		}
	}

	// Enable direct IP; on failure (e.g. unsupported flag) warn and continue.
	if err := env.RS.EnableDirectIP(ctx); err != nil {
		res.Actions = append(res.Actions, "direct IP must be enabled manually (UI)")
	} else {
		res.Actions = append(res.Actions, "enabled direct IP")
	}

	// Finalise config.
	cfg.ConfigVersion = config.SchemaVersion
	cfg.Device.RustDeskVersion = pinned
	if cfg.Device.Alias == "" {
		cfg.Device.Alias = "kendali"
	}
	if cfg.Terminal.Token == "" {
		tok, err := config.NewToken()
		if err != nil {
			return res, fmt.Errorf("install: generate token: %w", err)
		}
		cfg.Terminal.Token = tok
	}
	if cfg.Terminal.Bind == "" {
		cfg.Terminal.Bind = "localhost"
	}
	if cfg.Terminal.Port == 0 {
		cfg.Terminal.Port = 47654
	}
	if cfg.Terminal.IdleTimeout == 0 {
		cfg.Terminal.IdleTimeout = 30 * time.Minute
	}

	if !configExisted {
		changed = true
		res.Actions = append(res.Actions, "created rc.yaml")
	}

	if err := config.Save(env.ConfigPath, cfg); err != nil {
		return res, fmt.Errorf("install: save config: %w", err)
	}
	env.Config = cfg

	res.Changed = changed && !adopted
	return res, nil
}
