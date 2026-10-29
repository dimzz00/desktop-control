// SPDX-License-Identifier: AGPL-3.0-or-later

package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dimzz00/kendali/internal/config"
	"github.com/dimzz00/kendali/internal/deps"
	"github.com/dimzz00/kendali/internal/platform"
	"github.com/dimzz00/kendali/internal/rustdesk"
)

type fakeRunner struct {
	out       map[string][]byte
	err       map[string]error
	installed bool
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
	key := name
	for _, a := range args {
		key += " " + a
	}
	if strings.HasPrefix(key, "dpkg -i ") {
		f.installed = true
	}
	if key == "rustdesk --version" {
		if f.installed {
			return []byte("rustdesk 1.4.2\n"), nil, nil
		}
		if e, ok := f.err[key]; ok {
			return nil, []byte("err"), e
		}
		if v, ok := f.out[key]; ok {
			return v, nil, nil
		}
		return nil, nil, errors.New("unexpected: " + key)
	}
	if key == "systemctl is-active rustdesk" {
		if f.installed {
			return []byte("active\n"), nil, nil
		}
	}
	if e, ok := f.err[key]; ok {
		return nil, []byte("err"), e
	}
	if v, ok := f.out[key]; ok {
		return v, nil, nil
	}
	if strings.HasPrefix(key, "rustdesk --password ") {
		return []byte("ok\n"), nil, nil
	}
	if key == "rustdesk --help" {
		return []byte("Usage: rustdesk [--password <pw>]\n"), nil, nil
	}
	if key == "rustdesk --option direct-server Y" {
		return []byte("ok\n"), nil, nil
	}
	if strings.HasPrefix(key, "dpkg -i ") {
		if e, ok := f.err[key]; ok {
			return nil, []byte("err"), e
		}
		return []byte("ok\n"), nil, nil
	}
	if strings.HasPrefix(key, "msiexec ") {
		return []byte("ok\n"), nil, nil
	}
	return nil, nil, errors.New("unexpected: " + key)
}

func baseEnv(t *testing.T) (Env, string) {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	_ = os.MkdirAll(root, 0o755)
	layout := platform.Layout{
		Root:       root,
		Releases:   filepath.Join(root, "releases"),
		ConfigFile: filepath.Join(dir, "rc.yaml"),
		LogDir:     filepath.Join(root, "logs"),
		BinDir:     filepath.Join(root, "current"),
	}
	_ = os.MkdirAll(layout.Releases, 0o755)
	runner := &fakeRunner{out: map[string][]byte{}, err: map[string]error{}}
	mgr := rustdesk.Manager{Run: runner, GOOS: "linux"}
	return Env{
		OS:         "linux",
		Run:        runner,
		Layout:     layout,
		ConfigPath: layout.ConfigFile,
		Deps: []deps.Entry{{Name: "rustdesk", Version: "1.4.2",
			URL: "https://example.com/rustdesk.deb", SHA256: "a"}},
		Fetch: func(ctx context.Context, url, sha, dest string) (string, error) {
			p := filepath.Join(dest, "rustdesk.deb")
			_ = os.WriteFile(p, []byte("deb"), 0o644)
			return p, nil
		},
		RS: mgr,
	}, dir
}

func setInstalled(runner *fakeRunner, version string) {
	runner.out["rustdesk --version"] = []byte("rustdesk " + version + "\n")
	runner.out["systemctl is-active rustdesk"] = []byte("active\n")
	runner.out["rustdesk --help"] = []byte("Usage: rustdesk [--password <pw>]\n")
	runner.out["rustdesk --option direct-server Y"] = []byte("ok\n")
}

func TestRunIsIdempotent(t *testing.T) {
	env, _ := baseEnv(t)
	runner := env.Run.(*fakeRunner)
	setInstalled(runner, "1.4.2")

	r1, err := Run(context.Background(), env, Options{})
	if err != nil || !r1.Changed {
		t.Fatalf("r1=%+v err=%v", r1, err)
	}
	// second run must be no-op
	r2, err := Run(context.Background(), env, Options{})
	if err != nil || r2.Changed {
		t.Fatalf("r2=%+v err=%v", r2, err)
	}
}

func TestRunRefusesDifferentExistingVersion(t *testing.T) {
	env, _ := baseEnv(t)
	runner := env.Run.(*fakeRunner)
	setInstalled(runner, "1.4.1")
	_, err := Run(context.Background(), env, Options{})
	if !errors.Is(err, ErrExistingMismatch) {
		t.Fatalf("got %v", err)
	}
}

func TestRunAdoptsWhenAsked(t *testing.T) {
	env, _ := baseEnv(t)
	runner := env.Run.(*fakeRunner)
	setInstalled(runner, "1.4.1")
	r, err := Run(context.Background(), env, Options{AdoptExisting: true})
	if err != nil || r.Changed {
		t.Fatalf("r=%+v err=%v", r, err)
	}
	c, err := config.Load(env.ConfigPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c.Device.RustDeskVersion != "1.4.1" {
		t.Fatalf("version = %s", c.Device.RustDeskVersion)
	}
}

func TestRunLeavesConfigUntouchedWhenDownloadFails(t *testing.T) {
	env, _ := baseEnv(t)
	runner := env.Run.(*fakeRunner)
	// not installed -> need install -> fetch
	delete(runner.out, "rustdesk --version")
	runner.err["rustdesk --version"] = errors.New("not found")
	// pre-existing config
	cfg := config.Default()
	_ = config.Save(env.ConfigPath, cfg)
	before, _ := os.ReadFile(env.ConfigPath)

	env.Fetch = func(ctx context.Context, url, sha, dest string) (string, error) {
		return "", errors.New("boom")
	}
	_, err := Run(context.Background(), env, Options{})
	if err == nil {
		t.Fatal("expected error")
	}
	after, _ := os.ReadFile(env.ConfigPath)
	if string(before) != string(after) {
		t.Fatalf("config changed on fetch failure")
	}
}

func TestRunInstallsWhenMissing(t *testing.T) {
	env, _ := baseEnv(t)
	runner := env.Run.(*fakeRunner)
	// not installed
	runner.err["rustdesk --version"] = os.ErrNotExist
	delete(runner.out, "rustdesk --version")
	runner.out["systemctl is-active rustdesk"] = []byte("active\n")
	r, err := Run(context.Background(), env, Options{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if !r.Changed {
		t.Fatalf("expected changed")
	}
	_ = runner
}

func TestRunInstallPackageFailure(t *testing.T) {
	env, _ := baseEnv(t)
	runner := env.Run.(*fakeRunner)
	runner.err["rustdesk --version"] = os.ErrNotExist
	delete(runner.out, "rustdesk --version")
	// make dpkg fail
	runner.err["dpkg -i "+filepath.Join(env.Layout.Releases, "rustdesk.deb")] = errors.New("dpkg fail")
	// ensure fetch succeeds
	env.Fetch = func(ctx context.Context, url, sha, dest string) (string, error) {
		p := filepath.Join(dest, "rustdesk.deb")
		_ = os.WriteFile(p, []byte("deb"), 0o644)
		return p, nil
	}
	_, err := Run(context.Background(), env, Options{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunCreatesConfigOnce(t *testing.T) {
	env, _ := baseEnv(t)
	runner := env.Run.(*fakeRunner)
	setInstalled(runner, "1.4.2")
	// no config exists
	r, err := Run(context.Background(), env, Options{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	_ = r
	c, err := config.Load(env.ConfigPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(c.Terminal.Token) != 64 {
		t.Fatalf("token len = %d", len(c.Terminal.Token))
	}
	info, _ := os.Stat(env.ConfigPath)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
}
