// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunInstallMissingDeps(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(old) }()
	var stdout, stderr bytes.Buffer
	code := runInstall(context.Background(), nil, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected non-zero, got 0")
	}
}

func TestRunUninstall(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runUninstall(nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	_ = filepath.Base
}
