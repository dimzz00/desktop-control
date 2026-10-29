// SPDX-License-Identifier: AGPL-3.0-or-later

package terminal

import (
	"testing"
	"time"
)

func TestAuthBadToken(t *testing.T) {
	a := NewAuth("secret")
	ok, err := a.Check("1.2.3.4", "wrong")
	_ = ok
	_ = err
	if ok || err != ErrBadToken {
		t.Fatalf("got %v %v", ok, err)
	}
}

func TestAuthConstantTimeCorrect(t *testing.T) {
	a := NewAuth("secret")
	ok, err := a.Check("1.2.3.4", "secret")
	if !ok || err != nil {
		t.Fatalf("got %v %v", ok, err)
	}
}

func TestAuthRateLimit(t *testing.T) {
	a := NewAuth("secret")
	a.Limit = 2
	a.Window = time.Hour
	for i := 0; i < 2; i++ {
		_, _ = a.Check("1.2.3.4", "wrong")
	}
	_, err := a.Check("1.2.3.4", "secret")
	if err != ErrRateLimited {
		t.Fatalf("got %v, want rate limited", err)
	}
}

func TestAuthLockout(t *testing.T) {
	a := NewAuth("secret")
	a.Limit = 100
	a.Lockout = time.Second
	for i := 0; i < 10; i++ {
		_, _ = a.Check("1.2.3.4", "wrong")
	}
	_, err := a.Check("1.2.3.4", "secret")
	if err != ErrLockedOut {
		t.Fatalf("got %v, want locked out", err)
	}
}
