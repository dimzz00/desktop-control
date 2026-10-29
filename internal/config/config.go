// SPDX-License-Identifier: AGPL-3.0-or-later

// Package config reads, validates, and atomically persists the kendali
// configuration file (rc.yaml).
//
// The file carries its own schema number, config_version, independent of the
// application version. A file written by a newer release is rejected with
// ErrFutureSchema instead of being guessed at.
package config

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// SchemaVersion is the configuration schema this build understands.
const SchemaVersion = 1

// ErrFutureSchema reports a configuration file produced by a newer kendali
// release. Callers match it with errors.Is.
var ErrFutureSchema = errors.New("config: schema newer than supported")

var (
	aliasPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)
	tokenPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// Config is the root of rc.yaml.
type Config struct {
	ConfigVersion int      `yaml:"config_version"`
	Device        Device   `yaml:"device"`
	Terminal      Terminal `yaml:"terminal"`
}

// Device holds the identity of this host in the RustDesk fleet.
type Device struct {
	Alias           string `yaml:"alias"`
	RustDeskVersion string `yaml:"rustdesk_version"`
	Password        string `yaml:"password"`
}

// Terminal configures the local HTTP terminal endpoint.
type Terminal struct {
	Bind        string        `yaml:"bind"`
	Port        int           `yaml:"port"`
	Token       string        `yaml:"token"`
	IdleTimeout time.Duration `yaml:"idle_timeout"`
}

// Default returns a complete, valid configuration with a freshly minted
// terminal token.
func Default() *Config {
	token, err := NewToken()
	if err != nil {
		// crypto/rand failing means the host cannot produce secrets; there is
		// no safe configuration to fall back to.
		panic("config: cannot generate default token: " + err.Error())
	}
	return &Config{
		ConfigVersion: SchemaVersion,
		Device:        Device{Alias: "kendali"},
		Terminal: Terminal{
			Bind:        "localhost",
			Port:        47654,
			Token:       token,
			IdleTimeout: 30 * time.Minute,
		},
	}
}

// Load reads and validates the configuration at path. A missing file is
// reported as an error wrapping os.ErrNotExist.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	var file fileConfig
	if err := dec.Decode(&file); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("config: %s is empty", path)
		}
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	var extra yaml.Node
	switch err := dec.Decode(&extra); {
	case err == nil:
		return nil, fmt.Errorf("config: parse %s: unexpected trailing YAML document", path)
	case !errors.Is(err, io.EOF):
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	switch {
	case file.ConfigVersion > SchemaVersion:
		return nil, fmt.Errorf("%w: found %d, supported %d", ErrFutureSchema, file.ConfigVersion, SchemaVersion)
	case file.ConfigVersion < 1:
		return nil, fmt.Errorf("config: invalid config_version %d (minimum 1)", file.ConfigVersion)
	}

	c, err := file.toConfig()
	if err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("config: %s: %w", path, err)
	}
	return c, nil
}

// Save validates nothing; it serializes c to path atomically. The data is
// written to path+".tmp" with mode 0600, synced, then renamed over path, so a
// failed Save leaves any existing file untouched.
func Save(path string, c *Config) error {
	if c == nil {
		return errors.New("config: cannot save a nil config")
	}

	data, err := yaml.Marshal(fileConfigFrom(c))
	if err != nil {
		return fmt.Errorf("config: encode: %w", err)
	}

	tmp := path + ".tmp"
	if err := writeFileSync(tmp, data); err != nil {
		removeTemp(tmp)
		return fmt.Errorf("config: write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		removeTemp(tmp)
		return fmt.Errorf("config: replace %s: %w", path, err)
	}
	return nil
}

// Validate reports whether c is safe to run with. It normalizes the alias in
// place before checking it.
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config: nil config")
	}
	c.Normalize()

	if c.ConfigVersion != SchemaVersion {
		return fmt.Errorf("config: config_version %d is not supported (want %d)", c.ConfigVersion, SchemaVersion)
	}
	if !aliasPattern.MatchString(c.Device.Alias) {
		return fmt.Errorf("config: device.alias %q must match %s", c.Device.Alias, aliasPattern)
	}
	switch c.Terminal.Bind {
	case "localhost", "tailnet":
	default:
		return fmt.Errorf("config: terminal.bind %q must be %q or %q", c.Terminal.Bind, "localhost", "tailnet")
	}
	if c.Terminal.Port < 1024 || c.Terminal.Port > 65535 {
		return fmt.Errorf("config: terminal.port %d must be between 1024 and 65535", c.Terminal.Port)
	}
	if !tokenPattern.MatchString(c.Terminal.Token) {
		return errors.New("config: terminal.token must be 64 lowercase hex characters")
	}
	if c.Terminal.IdleTimeout <= 0 {
		return fmt.Errorf("config: terminal.idle_timeout %v must be positive", c.Terminal.IdleTimeout)
	}
	return nil
}

// Normalize trims surrounding whitespace from the alias and lowercases it.
func (c *Config) Normalize() {
	c.Device.Alias = strings.ToLower(strings.TrimSpace(c.Device.Alias))
}

// NewToken returns 32 cryptographically random bytes as 64 lowercase hex
// characters.
func NewToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("config: generate token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// fileConfig mirrors Config for YAML (de)serialization. idle_timeout is stored
// as a human-readable duration such as "30m" rather than a raw nanosecond
// count, so the file stays editable by hand.
type fileConfig struct {
	ConfigVersion int          `yaml:"config_version"`
	Device        Device       `yaml:"device"`
	Terminal      fileTerminal `yaml:"terminal"`
}

type fileTerminal struct {
	Bind        string `yaml:"bind"`
	Port        int    `yaml:"port"`
	Token       string `yaml:"token"`
	IdleTimeout string `yaml:"idle_timeout"`
}

func fileConfigFrom(c *Config) fileConfig {
	return fileConfig{
		ConfigVersion: c.ConfigVersion,
		Device:        c.Device,
		Terminal: fileTerminal{
			Bind:        c.Terminal.Bind,
			Port:        c.Terminal.Port,
			Token:       c.Terminal.Token,
			IdleTimeout: formatDuration(c.Terminal.IdleTimeout),
		},
	}
}

func (f fileConfig) toConfig() (*Config, error) {
	var idle time.Duration
	if f.Terminal.IdleTimeout != "" {
		parsed, err := time.ParseDuration(f.Terminal.IdleTimeout)
		if err != nil {
			return nil, fmt.Errorf("terminal.idle_timeout: %w", err)
		}
		idle = parsed
	}
	return &Config{
		ConfigVersion: f.ConfigVersion,
		Device:        f.Device,
		Terminal: Terminal{
			Bind:        f.Terminal.Bind,
			Port:        f.Terminal.Port,
			Token:       f.Terminal.Token,
			IdleTimeout: idle,
		},
	}, nil
}

// formatDuration renders d in the shortest exact unit: "30m", "90s", or
// time.Duration's own String for anything finer.
func formatDuration(d time.Duration) string {
	switch {
	case d == 0:
		return "0s"
	case d%time.Minute == 0:
		return strconv.FormatInt(int64(d/time.Minute), 10) + "m"
	case d%time.Second == 0:
		return strconv.FormatInt(int64(d/time.Second), 10) + "s"
	default:
		return d.String()
	}
}

func writeFileSync(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// removeTemp deletes a leftover temp file, but never a directory that happens
// to occupy the same path.
func removeTemp(path string) {
	if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
		_ = os.Remove(path)
	}
}
