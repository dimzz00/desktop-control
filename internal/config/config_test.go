// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestLoadRejectsFutureSchema(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "rc.yaml")
	_ = os.WriteFile(p, []byte("config_version: 2\n"), 0o600)
	if _, err := Load(p); !errors.Is(err, ErrFutureSchema) {
		t.Fatalf("got %v", err)
	}
}

func TestLoadRejectsOldSchema(t *testing.T) {
	for _, version := range []string{"0", "-1"} {
		t.Run(version, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "rc.yaml")
			if err := os.WriteFile(p, []byte("config_version: "+version+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Load(p)
			if err == nil {
				t.Fatal("want error for old schema")
			}
			if errors.Is(err, ErrFutureSchema) {
				t.Fatalf("old schema must not report ErrFutureSchema: %v", err)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	c := Default()
	c.Device.Alias = "Bad Alias!"
	if c.Validate() == nil {
		t.Fatal("alias harus ditolak")
	}

	c = Default()
	c.Terminal.Port = 80
	if c.Validate() == nil {
		t.Fatal("port <1024 harus ditolak")
	}

	c = Default()
	if err := c.Validate(); err != nil {
		t.Fatalf("default harus valid: %v", err)
	}
}

func TestValidateRules(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{"valid default", func(*Config) {}, false},
		{"nil config", nil, true},
		{"alias trimmed and lowercased", func(c *Config) { c.Device.Alias = "  Kendali-Host  " }, false},
		{"alias uppercase", func(c *Config) { c.Device.Alias = "KENDALI" }, false},
		{"alias numeric start", func(c *Config) { c.Device.Alias = "1host" }, false},
		{"alias empty", func(c *Config) { c.Device.Alias = "" }, true},
		{"alias leading dash", func(c *Config) { c.Device.Alias = "-host" }, true},
		{"alias underscore", func(c *Config) { c.Device.Alias = "my_host" }, true},
		{"alias too long", func(c *Config) { c.Device.Alias = strings.Repeat("a", 33) }, true},
		{"alias max length", func(c *Config) { c.Device.Alias = strings.Repeat("a", 32) }, false},
		{"bind localhost", func(c *Config) { c.Terminal.Bind = "localhost" }, false},
		{"bind tailnet", func(c *Config) { c.Terminal.Bind = "tailnet" }, false},
		{"bind other", func(c *Config) { c.Terminal.Bind = "0.0.0.0" }, true},
		{"bind empty", func(c *Config) { c.Terminal.Bind = "" }, true},
		{"port low", func(c *Config) { c.Terminal.Port = 1023 }, true},
		{"port minimum", func(c *Config) { c.Terminal.Port = 1024 }, false},
		{"port maximum", func(c *Config) { c.Terminal.Port = 65535 }, false},
		{"port high", func(c *Config) { c.Terminal.Port = 65536 }, true},
		{"token uppercase", func(c *Config) { c.Terminal.Token = strings.ToUpper(c.Terminal.Token) }, true},
		{"token short", func(c *Config) { c.Terminal.Token = c.Terminal.Token[:63] }, true},
		{"token empty", func(c *Config) { c.Terminal.Token = "" }, true},
		{"idle timeout zero", func(c *Config) { c.Terminal.IdleTimeout = 0 }, true},
		{"idle timeout negative", func(c *Config) { c.Terminal.IdleTimeout = -time.Minute }, true},
		{"schema version zero", func(c *Config) { c.ConfigVersion = 0 }, true},
		{"schema version future", func(c *Config) { c.ConfigVersion = SchemaVersion + 1 }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Default()
			if tt.mutate == nil {
				var nilConfig *Config
				if err := nilConfig.Validate(); err == nil {
					t.Fatal("want error for nil config")
				}
				return
			}
			tt.mutate(c)
			err := c.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("want error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("want nil, got %v", err)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	c := Default()
	c.Device.Alias = "  Kendali-Host  "
	c.Normalize()
	if got, want := c.Device.Alias, "kendali-host"; got != want {
		t.Fatalf("alias = %q, want %q", got, want)
	}
}

func TestSaveIsAtomicAndPrivate(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "rc.yaml")

	c := Default()
	c.Device.Alias = "kendali-test"
	if err := Save(p, c); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(p)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 0600", got)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "rc.yaml" {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("directory entries = %v, want only rc.yaml", names)
	}

	loaded, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.ConfigVersion != c.ConfigVersion ||
		loaded.Device != c.Device ||
		loaded.Terminal != c.Terminal {
		t.Fatalf("round trip = %+v, want %+v", loaded, c)
	}
}

func TestSaveFailureLeavesExistingFileUntouched(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "rc.yaml")
	if err := Save(p, Default()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	before, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}

	// Occupy the temporary path with a directory so the atomic write cannot
	// replace it and must fail cleanly.
	tmp := p + ".tmp"
	if err := os.Mkdir(tmp, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := Save(p, Default()); err == nil {
		t.Fatal("Save over an occupied temp path must fail")
	}

	after, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("existing config disappeared: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("existing config changed\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if _, err := os.Stat(tmp); err != nil {
		t.Fatalf("temp path should be left alone: %v", err)
	}
}

func TestSaveRejectsUnwritableDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "rc.yaml")
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	if err := Save(p, Default()); err == nil {
		t.Fatal("Save into a read-only directory must fail")
	}
	if _, err := os.Stat(p + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temp file left behind: %v", err)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "absent.yaml"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("got %v, want os.ErrNotExist", err)
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	tests := map[string]string{
		"top level":  "config_version: 1\nbogus: true\n",
		"device":     "config_version: 1\ndevice:\n  bogus: true\n",
		"terminal":   "config_version: 1\nterminal:\n  bogus: true\n",
		"duplicated": "config_version: 1\nconfig_version: 1\n",
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "rc.yaml")
			if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(p); err == nil {
				t.Fatal("unknown or duplicate keys must be rejected")
			}
		})
	}
}

func TestLoadRejectsMalformedInput(t *testing.T) {
	tests := map[string]string{
		"empty":            "",
		"garbage":          ":\n  - [\n",
		"wrong type":       "config_version: one\n",
		"trailing doc":     "config_version: 1\n---\nconfig_version: 1\n",
		"invalid port":     "config_version: 1\nterminal:\n  port: 80\n",
		"invalid duration": "config_version: 1\nterminal:\n  idle_timeout: soon\n",
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "rc.yaml")
			if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(p); err == nil {
				t.Fatal("want error")
			}
		})
	}
}

func TestSaveWritesReadableDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{90 * time.Second, "90s"},
		{30 * time.Minute, "30m"},
		{1500 * time.Millisecond, "1.5s"},
		{0, "0s"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "rc.yaml")
			c := Default()
			c.Terminal.IdleTimeout = tt.duration
			if err := Save(p, c); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			pattern := regexp.MustCompile(`(?m)^\s+idle_timeout: ` + regexp.QuoteMeta(tt.want) + `$`)
			if !pattern.Match(raw) {
				t.Fatalf("idle_timeout %v should be %q, got:\n%s", tt.duration, tt.want, raw)
			}
			loaded, err := Load(p)
			if err != nil {
				// A zero timeout is rejected by Validate, but parsing must
				// still have succeeded far enough to report that.
				if tt.duration == 0 {
					return
				}
				t.Fatal(err)
			}
			if loaded.Terminal.IdleTimeout != tt.duration {
				t.Fatalf("idle timeout = %v, want %v", loaded.Terminal.IdleTimeout, tt.duration)
			}
		})
	}
}

func TestSaveRejectsNilConfig(t *testing.T) {
	if err := Save(filepath.Join(t.TempDir(), "rc.yaml"), nil); err == nil {
		t.Fatal("Save(nil) must fail")
	}
}

func TestNewToken(t *testing.T) {
	first, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(first) {
		t.Fatalf("token %q is not 64 lowercase hex characters", first)
	}
	second, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	if first == second {
		t.Fatal("two tokens must differ")
	}
}

func TestDefault(t *testing.T) {
	c := Default()
	if c.ConfigVersion != SchemaVersion {
		t.Fatalf("ConfigVersion = %d, want %d", c.ConfigVersion, SchemaVersion)
	}
	if c.Device.Alias != "kendali" {
		t.Fatalf("alias = %q, want kendali", c.Device.Alias)
	}
	if c.Terminal.Bind != "localhost" || c.Terminal.Port != 47654 {
		t.Fatalf("terminal = %+v", c.Terminal)
	}
	if c.Terminal.IdleTimeout != 30*time.Minute {
		t.Fatalf("idle timeout = %v", c.Terminal.IdleTimeout)
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("default must be valid: %v", err)
	}
	if other := Default(); other.Terminal.Token == c.Terminal.Token {
		t.Fatal("each Default must mint a fresh token")
	}
}

func FuzzConfigParse(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("config_version: 1\n"))
	f.Add([]byte("config_version: 2\n"))
	f.Add([]byte("config_version: 1\ndevice:\n  alias: kendali\n"))
	f.Add([]byte("---\n\x00\xff"))

	f.Fuzz(func(t *testing.T, raw []byte) {
		// Each run gets a private directory so parallel fuzz workers never
		// share files; parsing is entirely local (no network access).
		p := filepath.Join(t.TempDir(), "rc.yaml")
		if err := os.WriteFile(p, raw, 0o600); err != nil {
			t.Fatal(err)
		}
		_, _ = Load(p)
	})
}
