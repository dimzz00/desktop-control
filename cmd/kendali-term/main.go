// SPDX-License-Identifier: AGPL-3.0-or-later

// Command kendali-term is the terminal sidecar for kendali.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dimzz00/kendali/internal/config"
	"github.com/dimzz00/kendali/internal/terminal"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stdout, terminal.VersionString())
		return 0
	}
	switch args[0] {
	case "version":
		_, _ = fmt.Fprintln(stdout, terminal.VersionString())
		return 0
	case "serve":
		fs := flag.NewFlagSet("serve", flag.ContinueOnError)
		fs.SetOutput(stderr)
		bind := fs.String("bind", "localhost", "bind address mode")
		port := fs.Int("port", 47654, "port")
		token := fs.String("token", "", "auth token")
		idle := fs.Duration("idle", 30*time.Minute, "idle timeout")
		configPath := fs.String("config", "", "rc.yaml path")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if *configPath != "" {
			if c, err := config.Load(*configPath); err == nil {
				if *bind == "localhost" && c.Terminal.Bind != "" {
					*bind = c.Terminal.Bind
				}
				if *port == 47654 && c.Terminal.Port != 0 {
					*port = c.Terminal.Port
				}
				if *token == "" {
					*token = c.Terminal.Token
				}
				if *idle == 30*time.Minute && c.Terminal.IdleTimeout != 0 {
					*idle = c.Terminal.IdleTimeout
				}
			}
		}
		cfg := ServerConfig{Bind: *bind, Port: *port, Token: *token, Idle: *idle}
		if err := Serve(context.Background(), cfg); err != nil {
			_, _ = fmt.Fprintf(stderr, "kendali-term: %v\n", err)
			return 1
		}
		return 0
	}
	_, _ = fmt.Fprintf(stderr, "unknown command %q\n", args[0])
	return 2
}
