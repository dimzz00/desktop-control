// SPDX-License-Identifier: AGPL-3.0-or-later

package tailnet

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	"github.com/dimzz00/kendali/internal/platform"
)

type fakeRunner struct {
	out map[string][]byte
	err map[string]error
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
	key := name
	for _, a := range args {
		key += " " + a
	}
	if e, ok := f.err[key]; ok {
		return nil, []byte("not logged in"), e
	}
	if v, ok := f.out[key]; ok {
		return v, nil, nil
	}
	return nil, nil, exec.ErrNotFound
}

func TestDetectNotInstalled(t *testing.T) {
	f := &fakeRunner{err: map[string]error{"tailscale version": exec.ErrNotFound}}
	st, err := Detect(context.Background(), f)
	if err != nil || st.Installed || st.Running {
		t.Fatalf("got %+v %v", st, err)
	}
}

func TestDetectLoggedOut(t *testing.T) {
	f := &fakeRunner{
		out: map[string][]byte{"tailscale version": []byte("1.50.0\n")},
		err: map[string]error{"tailscale ip -4": errors.New("not logged in")},
	}
	st, err := Detect(context.Background(), f)
	if err != nil || !st.Installed || st.Running || st.IPv4 != nil {
		t.Fatalf("got %+v %v", st, err)
	}
}

func TestDetectRunning(t *testing.T) {
	f := &fakeRunner{out: map[string][]byte{
		"tailscale version": []byte("1.50.0\n"),
		"tailscale ip -4":   []byte("100.101.102.103\n"),
	}}
	st, err := Detect(context.Background(), f)
	if err != nil || !st.Installed || !st.Running || st.IPv4.String() != "100.101.102.103" {
		t.Fatalf("got %+v %v", st, err)
	}
}

func TestDetectRejectsNonTailnetIP(t *testing.T) {
	f := &fakeRunner{out: map[string][]byte{
		"tailscale version": []byte("1.50.0\n"),
		"tailscale ip -4":   []byte("192.168.1.5\n"),
	}}
	st, err := Detect(context.Background(), f)
	if err != nil || !st.Installed || st.Running || st.IPv4 != nil {
		t.Fatalf("got %+v %v", st, err)
	}
}

var _ = platform.Runner(nil)
