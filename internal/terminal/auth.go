// SPDX-License-Identifier: AGPL-3.0-or-later

package terminal

import (
	"crypto/subtle"
	"sync"
	"time"
)

// Auth enforces token validation with rate limiting and lockout.
type Auth struct {
	Token   string
	Limit   int           // attempts per window
	Window  time.Duration // rate-limit window
	Lockout time.Duration // lockout after too many failures

	mu       sync.Mutex
	attempts map[string][]time.Time
	failures map[string]int
	locked   map[string]time.Time
}

func NewAuth(token string) *Auth {
	return &Auth{
		Token:    token,
		Limit:    5,
		Window:   time.Minute,
		Lockout:  15 * time.Minute,
		attempts: map[string][]time.Time{},
		failures: map[string]int{},
		locked:   map[string]time.Time{},
	}
}

// Check returns true if token matches and rate limit allows.
func (a *Auth) Check(ip, token string) (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	if until, ok := a.locked[ip]; ok && now.Before(until) {
		return false, ErrLockedOut
	}

	// prune old attempts
	var recent []time.Time
	for _, t := range a.attempts[ip] {
		if now.Sub(t) < a.Window {
			recent = append(recent, t)
		}
	}
	a.attempts[ip] = recent
	if len(recent) >= a.Limit {
		return false, ErrRateLimited
	}
	a.attempts[ip] = append(a.attempts[ip], now)

	if subtle.ConstantTimeCompare([]byte(token), []byte(a.Token)) != 1 {
		a.failures[ip]++
		if a.failures[ip] >= 10 {
			a.locked[ip] = now.Add(a.Lockout)
			a.failures[ip] = 0
		}
		return false, ErrBadToken
	}
	a.failures[ip] = 0
	return true, nil
}

// Reset clears state for an IP.
func (a *Auth) Reset(ip string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.attempts, ip)
	delete(a.failures, ip)
	delete(a.locked, ip)
}

var (
	ErrBadToken    = &authErr{"bad token"}
	ErrRateLimited = &authErr{"rate limited"}
	ErrLockedOut   = &authErr{"locked out"}
)

type authErr struct{ msg string }

func (e *authErr) Error() string { return "auth: " + e.msg }
