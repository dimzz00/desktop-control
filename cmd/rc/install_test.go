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

func TestRunDoctorSmoke(t *testing.T) {
	var stdout, stderr bytes.Buffer
	_ = runDoctor(context.Background(), nil, &stdout, &stderr)
}

func TestRunLogsSmoke(t *testing.T) {
	var stdout, stderr bytes.Buffer
	_ = runLogs(context.Background(), nil, &stdout, &stderr)
}

func TestRunTokenMissingConfig(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runToken(context.Background(), nil, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected non-zero without config")
	}
}

func TestRunTermMissingConfig(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runTerm(context.Background(), nil, &stdout, &stderr)
	_ = code
}

func TestRunTermStatusSmoke(t *testing.T) {
	var stdout, stderr bytes.Buffer
	_ = runTerm(context.Background(), []string{"status"}, &stdout, &stderr)
}

func TestRunStatusSmoke(t *testing.T) {
	var stdout, stderr bytes.Buffer
	_ = runStatus(context.Background(), nil, &stdout, &stderr)
}

func TestRunInstallSmoke(t *testing.T) {
	var stdout, stderr bytes.Buffer
	_ = runInstall(context.Background(), nil, &stdout, &stderr)
}
