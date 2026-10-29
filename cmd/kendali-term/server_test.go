// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestServeSmoke(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := ServerConfig{Bind: "localhost", Port: 18923, Token: "tok", Idle: time.Minute}
	go func() { _ = Serve(ctx, cfg) }()
	time.Sleep(300 * time.Millisecond)

	// static
	resp, err := http.Get("http://127.0.0.1:18923/")
	if err == nil {
		_ = resp.Body.Close()
	}
	// unauthorized ws
	resp2, err2 := http.Get("http://127.0.0.1:18923/ws")
	if err2 == nil {
		_ = resp2.Body.Close()
	}
	// authorized (token in query) - triggers pty path
	resp3, err3 := http.Get("http://127.0.0.1:18923/ws?token=tok")
	if err3 == nil {
		_ = resp3.Body.Close()
	}

	cancel()
	time.Sleep(100 * time.Millisecond)
}

func TestBindAddr(t *testing.T) {
	if bindAddr(ServerConfig{Bind: "localhost"}) != "127.0.0.1" {
		t.Fatal("localhost bind wrong")
	}
}
