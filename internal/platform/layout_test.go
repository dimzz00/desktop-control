// SPDX-License-Identifier: AGPL-3.0-or-later

package platform

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestForOSLayout(t *testing.T) {
	l := ForOS("linux")
	if l.Root != "/opt/kendali" || l.Releases != "/opt/kendali/releases" ||
		l.ConfigFile != "/etc/kendali/rc.yaml" || l.LogDir != "/var/log/kendali" ||
		l.BinDir != "/opt/kendali/current" {
		t.Fatalf("linux layout wrong: %+v", l)
	}
	w := ForOS("windows")
	if !strings.HasSuffix(w.ConfigFile, `\kendali\rc.yaml`) {
		t.Fatalf("windows config path wrong: %+v", w)
	}
	if w.Root == "" || w.Releases == "" || w.LogDir == "" || w.BinDir == "" {
		t.Fatalf("windows layout incomplete: %+v", w)
	}
}

func TestForOSUnknownDefaultsToLinux(t *testing.T) {
	u := ForOS("plan9")
	if u.ConfigFile != "/etc/kendali/rc.yaml" {
		t.Fatalf("unknown os should fall back to linux layout: %+v", u)
	}
}

func TestExecReportsExitCode(t *testing.T) {
	_, _, err := Exec{}.Run(context.Background(), "sh", "-c", "exit 3")
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("got %v, want *exec.ExitError", err)
	}
	if ee.ExitCode() != 3 {
		t.Fatalf("exit code = %d, want 3", ee.ExitCode())
	}
}

func TestExecCapturesStdoutStderr(t *testing.T) {
	out, errOut, err := Exec{}.Run(context.Background(), "sh", "-c", "echo hi; echo bye >&2")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if string(out) != "hi\n" {
		t.Fatalf("stdout = %q, want %q", out, "hi\n")
	}
	if string(errOut) != "bye\n" {
		t.Fatalf("stderr = %q, want %q", errOut, "bye\n")
	}
}

type fakeRunner struct {
	out    []byte
	errOut []byte
	err    error
}

func (f *fakeRunner) Run(_ context.Context, _ string, _ ...string) ([]byte, []byte, error) {
	return f.out, f.errOut, f.err
}

func TestIsElevatedLinux(t *testing.T) {
	cases := []struct {
		name string
		out  string
		err  error
		want bool
	}{
		{"root", "0\n", nil, true},
		{"user", "1000\n", nil, false},
		{"error", "", errors.New("boom"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := &fakeRunner{out: []byte(c.out), err: c.err}
			got, err := IsElevated(context.Background(), f)
			if c.err != nil {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}
