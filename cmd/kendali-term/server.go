// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/coder/websocket"

	"github.com/dimzz00/kendali/internal/platform"
	"github.com/dimzz00/kendali/internal/tailnet"
	"github.com/dimzz00/kendali/internal/terminal"
)

// ServerConfig configures kendali-term.
type ServerConfig struct {
	Bind  string
	Port  int
	Token string
	Idle  time.Duration
}

// Serve runs the terminal HTTP/WebSocket server until ctx is cancelled.
func Serve(ctx context.Context, cfg ServerConfig) error {
	if cfg.Idle == 0 {
		cfg.Idle = 30 * time.Minute
	}
	if cfg.Bind == "" {
		cfg.Bind = "localhost"
	}
	auth := terminal.NewAuth(cfg.Token)
	mgr := terminal.NewManager(cfg.Idle)

	mux := http.NewServeMux()

	// static assets
	sub, _ := fs.Sub(staticFiles, "static")
	mux.Handle("/", http.FileServer(http.FS(sub)))

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		// token from query or cookie
		token := r.URL.Query().Get("token")
		if token == "" {
			if c, err := r.Cookie("term_session"); err == nil {
				token = c.Value
			}
		}
		// if cookie session exists, accept
		sessionCookie, _ := r.Cookie("term_session")
		hasSession := sessionCookie != nil

		if !hasSession {
			ok, err := auth.Check(ip, token)
			if !ok {
				http.Error(w, "unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}
			// set session cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "term_session",
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})
		}

		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

		// start PTY
		pty, err := terminal.NewPTY()
		if err != nil {
			func() { _ = conn.Close(websocket.StatusInternalError, err.Error()) }()
			return
		}
		shell := osShell()
		if err := pty.Start(shell, nil); err != nil {
			func() { _ = conn.Close(websocket.StatusInternalError, err.Error()) }()
			return
		}
		sess, _ := mgr.Create(ctx, pty)
		defer mgr.Close(sess.ID)

		// pump output
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := pty.Read(buf)
				if n > 0 {
					_ = conn.Write(ctx, websocket.MessageText, buf[:n])
				}
				if err != nil {
					_ = conn.Close(websocket.StatusNormalClosure, "")
					return
				}
				mgr.Touch(sess.ID)
			}
		}()

		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			mgr.Touch(sess.ID)
			// handle resize JSON
			var msg struct {
				Type string `json:"type"`
				Cols int    `json:"cols"`
				Rows int    `json:"rows"`
			}
			if json.Unmarshal(data, &msg) == nil && msg.Type == "resize" {
				_ = pty.Resize(msg.Cols, msg.Rows)
				continue
			}
			_, _ = pty.Write(data)
		}
	})

	addr := bindAddr(cfg)
	httpServer := &http.Server{Addr: fmt.Sprintf("%s:%d", addr, cfg.Port), Handler: mux}

	go func() {
		<-ctx.Done()
		func() { _ = httpServer.Shutdown(context.Background()) }()
	}()

	// reap idle sessions periodically
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mgr.Reap()
			}
		}
	}()

	err := httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func bindAddr(cfg ServerConfig) string {
	switch cfg.Bind {
	case "tailnet":
		runner := platform.Exec{}
		st, err := tailnet.Detect(context.Background(), runner)
		if err == nil && st.Running {
			return st.IPv4.String()
		}
		return "0.0.0.0"
	default:
		return "127.0.0.1"
	}
}

func osShell() string {
	if runtime.GOOS == "windows" {
		if s := os.Getenv("COMSPEC"); s != "" {
			return s
		}
		return "cmd.exe"
	}
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	return "/bin/sh"
}
