// SPDX-License-Identifier: AGPL-3.0-or-later

package rustdesk

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

type fakeRunner struct {
	calls []string
	out   map[string][]byte
	err   error
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
	key := name
	if len(args) > 0 {
		key = name + " " + strings.Join(args, " ")
	}
	f.calls = append(f.calls, key)
	if f.err != nil && (f.out == nil || f.out[key] == nil) {
		// allow per-command errors via a sentinel map key "__err__:<key>"
		if v, ok := f.out["__err__:"+key]; ok {
			_ = v
		}
	}
	if v, ok := f.out[key]; ok {
		return v, nil, nil
	}
	if f.err != nil {
		return nil, []byte("not found"), f.err
	}
	return nil, nil, errors.New("unexpected command: " + key)
}

func TestDetectLinux(t *testing.T) {
	f := &fakeRunner{out: map[string][]byte{
		"rustdesk --version":           []byte("rustdesk 1.4.2\n"),
		"systemctl is-active rustdesk": []byte("active\n"),
	}}
	s, err := Manager{Run: f, GOOS: "linux"}.Detect(context.Background())
	if err != nil || !s.Installed || s.Version != "1.4.2" || !s.ServiceActive {
		t.Fatalf("%+v %v", s, err)
	}
}

func TestDetectMissingBinary(t *testing.T) {
	f := &fakeRunner{err: exec.ErrNotFound}
	s, err := Manager{Run: f, GOOS: "linux"}.Detect(context.Background())
	if err != nil || s.Installed {
		t.Fatalf("got %+v %v, want installed=false err=nil", s, err)
	}
	f2 := &fakeRunner{err: os.ErrNotExist}
	s2, err2 := Manager{Run: f2, GOOS: "linux"}.Detect(context.Background())
	if err2 != nil || s2.Installed {
		t.Fatalf("got %+v %v", s2, err2)
	}
}

func TestDetectServiceInactive(t *testing.T) {
	f := &fakeRunner{out: map[string][]byte{
		"rustdesk --version":           []byte("rustdesk 1.4.2\n"),
		"systemctl is-active rustdesk": []byte("inactive\n"),
	}, err: errors.New("inactive")}
	// override: service command returns error
	f2 := &fakeRunner{out: map[string][]byte{
		"rustdesk --version": []byte("rustdesk 1.4.2\n"),
	}}
	s, err := Manager{Run: f2, GOOS: "linux"}.Detect(context.Background())
	if err != nil || !s.Installed || s.ServiceActive {
		t.Fatalf("got %+v %v", s, err)
	}
	_ = f
}

func TestSupportsPasswordFlag(t *testing.T) {
	f := &fakeRunner{out: map[string][]byte{
		"rustdesk --help": []byte("Usage: rustdesk [--password <pw>]\n"),
	}}
	ok, err := Manager{Run: f, GOOS: "linux"}.SupportsPasswordFlag(context.Background())
	if err != nil || !ok {
		t.Fatalf("got %v %v, want true nil", ok, err)
	}

	f2 := &fakeRunner{out: map[string][]byte{
		"rustdesk --help": []byte("Usage: rustdesk\n"),
	}}
	ok2, err2 := Manager{Run: f2, GOOS: "linux"}.SupportsPasswordFlag(context.Background())
	if err2 != nil || ok2 {
		t.Fatalf("got %v %v, want false nil", ok2, err2)
	}
}

func TestSetPasswordUsesFlagWhenSupported(t *testing.T) {
	f := &fakeRunner{out: map[string][]byte{
		"rustdesk --help":            []byte("rustdesk [--password <pw>]\n"),
		"rustdesk --password secret": []byte("ok\n"),
	}}
	err := Manager{Run: f, GOOS: "linux"}.SetPassword(context.Background(), "secret")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	joined := strings.Join(f.calls, "\n")
	if !strings.Contains(joined, "--password secret") {
		t.Fatalf("calls = %v, want --password secret", f.calls)
	}
}

func TestSetPasswordRefusesSilentlyUnsupported(t *testing.T) {
	f := &fakeRunner{out: map[string][]byte{
		"rustdesk --help": []byte("Usage: rustdesk\n"),
	}}
	err := Manager{Run: f, GOOS: "linux"}.SetPassword(context.Background(), "secret")
	if err == nil || !strings.Contains(err.Error(), "manual") {
		t.Fatalf("got %v, want manual-steps error", err)
	}
}

func TestInstallPackagePerOS(t *testing.T) {
	fl := &fakeRunner{out: map[string][]byte{
		"dpkg -i a.deb": []byte("ok\n"),
	}}
	mgrL := Manager{Run: fl, GOOS: "linux"}
	if err := mgrL.InstallPackage(context.Background(), "a.deb"); err != nil {
		t.Fatalf("linux: %v", err)
	}
	if !containsCall(fl.calls, "dpkg -i a.deb") {
		t.Fatalf("linux calls = %v", fl.calls)
	}

	fw := &fakeRunner{out: map[string][]byte{
		"msiexec /i a.msi /qn /norestart": []byte("ok\n"),
	}}
	mgrW := Manager{Run: fw, GOOS: "windows"}
	if err := mgrW.InstallPackage(context.Background(), "a.msi"); err != nil {
		t.Fatalf("windows: %v", err)
	}
	if !containsCall(fw.calls, "msiexec /i a.msi /qn /norestart") {
		t.Fatalf("windows calls = %v", fw.calls)
	}
}

func TestDetectWindows(t *testing.T) {
	f := &fakeRunner{out: map[string][]byte{
		"rustdesk.exe --version": []byte("rustdesk 1.4.2\n"),
		"sc query RustDesk":      []byte("STATE: 4 RUNNING\n"),
	}}
	s, err := Manager{Run: f, GOOS: "windows"}.Detect(context.Background())
	if err != nil || !s.Installed || !s.ServiceActive {
		t.Fatalf("got %+v %v", s, err)
	}
}

func TestID(t *testing.T) {
	f := &fakeRunner{out: map[string][]byte{
		"rustdesk --get-id": []byte("123456789\n"),
	}}
	id, err := Manager{Run: f, GOOS: "linux"}.ID(context.Background())
	if err != nil || id != "123456789" {
		t.Fatalf("got %q %v", id, err)
	}
}

func TestEnableDirectIP(t *testing.T) {
	f := &fakeRunner{out: map[string][]byte{
		"rustdesk --option direct-server Y": []byte("ok\n"),
	}}
	mgrE := Manager{Run: f, GOOS: "linux"}
	if err := mgrE.EnableDirectIP(context.Background()); err != nil {
		t.Fatalf("err = %v", err)
	}

	f2 := &fakeRunner{err: errors.New("unknown option")}
	err := Manager{Run: f2, GOOS: "linux"}.EnableDirectIP(context.Background())
	if err == nil || !strings.Contains(err.Error(), "manual") {
		t.Fatalf("got %v, want manual-steps error", err)
	}
}

func containsCall(calls []string, want string) bool {
	for _, c := range calls {
		if c == want {
			return true
		}
	}
	return false
}
