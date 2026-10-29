// SPDX-License-Identifier: AGPL-3.0-or-later

package fetch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// checksum returns the lowercase hex SHA-256 of body.
func checksum(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

// withMaxBytes lowers the download cap for the duration of the test, so the
// size guard can be exercised without moving hundreds of megabytes.
func withMaxBytes(t *testing.T, limit int64) {
	t.Helper()
	old := maxArtifactBytes
	maxArtifactBytes = limit
	t.Cleanup(func() { maxArtifactBytes = old })
}

// dirFiles lists the names in dir, sorted by os.ReadDir.
func dirFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// requireEmptyDir fails when dir contains anything at all.
func requireEmptyDir(t *testing.T, dir string) {
	t.Helper()
	if names := dirFiles(t, dir); len(names) != 0 {
		t.Fatalf("destination directory must be empty, found %v", names)
	}
}

// serveBody starts a server that answers every request with body.
func serveBody(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestDownloadVerifiesChecksum(t *testing.T) {
	srv := serveBody(t, "abc")
	dir := t.TempDir()

	p, err := Download(context.Background(), srv.Client(), srv.URL+"/rustdesk.deb", checksum("abc"), dir)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if filepath.Dir(p) != dir || filepath.Base(p) != "rustdesk.deb" {
		t.Fatalf("Download path = %q, want %q", p, filepath.Join(dir, "rustdesk.deb"))
	}
	if b, err := os.ReadFile(p); err != nil || string(b) != "abc" {
		t.Fatalf("downloaded content = %q, %v; want %q", b, err, "abc")
	}
	if names := dirFiles(t, dir); len(names) != 1 || names[0] != "rustdesk.deb" {
		t.Fatalf("destination directory = %v, want only rustdesk.deb", names)
	}
}

func TestDownloadAcceptsUppercaseChecksum(t *testing.T) {
	srv := serveBody(t, "abc")
	dir := t.TempDir()

	p, err := Download(context.Background(), srv.Client(), srv.URL+"/rustdesk.deb", strings.ToUpper(checksum("abc")), dir)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if b, _ := os.ReadFile(p); string(b) != "abc" {
		t.Fatalf("content = %q, want %q", b, "abc")
	}
}

func TestDownloadRejectsWrongChecksum(t *testing.T) {
	srv := serveBody(t, "abc")
	dir := t.TempDir()

	p, err := Download(context.Background(), srv.Client(), srv.URL+"/rustdesk.deb", checksum("xyz"), dir)
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("error = %v, want ErrChecksumMismatch", err)
	}
	if p != "" {
		t.Fatalf("path = %q, want empty on failure", p)
	}
	if !strings.Contains(err.Error(), checksum("xyz")) {
		t.Fatalf("error %q must name the expected checksum", err)
	}
	if !strings.Contains(err.Error(), checksum("abc")) {
		t.Fatalf("error %q must name the actual checksum", err)
	}
	requireEmptyDir(t, dir)
}

func TestDownloadRejectsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	dir := t.TempDir()

	if _, err := Download(context.Background(), srv.Client(), srv.URL+"/rustdesk.deb", checksum("abc"), dir); err == nil {
		t.Fatal("want an error for HTTP 404")
	} else if !strings.Contains(err.Error(), strconv.Itoa(http.StatusNotFound)) {
		t.Fatalf("error %q must contain status code %d", err, http.StatusNotFound)
	}
	requireEmptyDir(t, dir)
}

// partialBodyServer writes a response that promises ten bytes but closes the
// connection after three, simulating a dropped download.
func partialBodyServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		conn, buf, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Errorf("hijack: %v", err)
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = buf.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 10\r\n\r\nabc")
		_ = buf.Flush()
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestDownloadRemovesPartialFileOnFailure(t *testing.T) {
	srv := partialBodyServer(t)
	dir := t.TempDir()

	p, err := Download(context.Background(), srv.Client(), srv.URL+"/rustdesk.deb", checksum("abcdefghij"), dir)
	if err == nil {
		t.Fatal("want an error for a truncated response")
	}
	if p != "" {
		t.Fatalf("path = %q, want empty on failure", p)
	}
	if names := dirFiles(t, dir); len(names) != 0 {
		t.Fatalf("no file may be left behind, found %v", names)
	}
}

func TestDownloadEmptyBody(t *testing.T) {
	srv := serveBody(t, "")
	dir := t.TempDir()

	p, err := Download(context.Background(), srv.Client(), srv.URL+"/empty.deb", checksum(""), dir)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("size = %d, want 0", info.Size())
	}
}

func TestDownloadRejectsOversizedBody(t *testing.T) {
	withMaxBytes(t, 16)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Flushing first makes the response chunked, so the client cannot
		// learn the length up front and the streaming guard must catch it.
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		_, _ = w.Write(bytes.Repeat([]byte("x"), 32))
	}))
	defer srv.Close()
	dir := t.TempDir()

	_, err := Download(context.Background(), srv.Client(), srv.URL+"/big.deb", checksum(strings.Repeat("x", 32)), dir)
	if err == nil {
		t.Fatal("want an error for an oversized body")
	}
	if !strings.Contains(err.Error(), "limit") {
		t.Fatalf("error %q must mention the size limit", err)
	}
	requireEmptyDir(t, dir)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDownloadRejectsOversizedContentLength(t *testing.T) {
	withMaxBytes(t, 16)
	client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    http.StatusOK,
			Status:        "200 OK",
			ContentLength: 17,
			Body:          io.NopCloser(strings.NewReader("")),
			Header:        make(http.Header),
		}, nil
	})}
	dir := t.TempDir()

	_, err := Download(context.Background(), client, "https://x.example/big.deb", checksum(""), dir)
	if err == nil {
		t.Fatal("want an error for a declared oversized body")
	}
	if !strings.Contains(err.Error(), "limit") {
		t.Fatalf("error %q must mention the size limit", err)
	}
	requireEmptyDir(t, dir)
}

func TestDownloadRejectsBadURLBase(t *testing.T) {
	srv := serveBody(t, "abc")
	urls := []string{
		"",
		"/",
		"/.",
		"/..",
		"https://x.example/",
		"https://x.example/..",
	}
	for _, rawURL := range urls {
		t.Run(rawURL, func(t *testing.T) {
			dir := t.TempDir()
			_, err := Download(context.Background(), srv.Client(), rawURL, checksum("abc"), dir)
			if err == nil {
				t.Fatalf("Download(%q) must fail", rawURL)
			}
			if !strings.Contains(err.Error(), "file name") {
				t.Fatalf("error %q must explain the unusable file name", err)
			}
			requireEmptyDir(t, dir)
		})
	}
}

func TestDownloadRejectsMalformedURL(t *testing.T) {
	srv := serveBody(t, "abc")
	dir := t.TempDir()

	_, err := Download(context.Background(), srv.Client(), "https://x.example/\x7f.deb", checksum("abc"), dir)
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("error = %v, want an invalid-URL error", err)
	}
	requireEmptyDir(t, dir)
}

func TestDownloadReplacesExistingFile(t *testing.T) {
	srv := serveBody(t, "baru")
	dir := t.TempDir()
	final := filepath.Join(dir, "rustdesk.deb")
	if err := os.WriteFile(final, []byte("lama"), 0o600); err != nil {
		t.Fatal(err)
	}

	p, err := Download(context.Background(), srv.Client(), srv.URL+"/rustdesk.deb", checksum("baru"), dir)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if p != final {
		t.Fatalf("path = %q, want %q", p, final)
	}
	if b, _ := os.ReadFile(final); string(b) != "baru" {
		t.Fatalf("content = %q, want %q", b, "baru")
	}
	if names := dirFiles(t, dir); len(names) != 1 {
		t.Fatalf("destination directory = %v, want only the final file", names)
	}
}

func TestDownloadPreservesExistingFileOnChecksumFailure(t *testing.T) {
	srv := serveBody(t, "abc")
	dir := t.TempDir()
	final := filepath.Join(dir, "rustdesk.deb")
	if err := os.WriteFile(final, []byte("lama"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Download(context.Background(), srv.Client(), srv.URL+"/rustdesk.deb", checksum("xyz"), dir)
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("error = %v, want ErrChecksumMismatch", err)
	}
	if b, _ := os.ReadFile(final); string(b) != "lama" {
		t.Fatalf("pre-existing file was modified: %q", b)
	}
	if names := dirFiles(t, dir); len(names) != 1 || names[0] != "rustdesk.deb" {
		t.Fatalf("destination directory = %v, want only the untouched final file", names)
	}
}

func TestDownloadRejectsNilClient(t *testing.T) {
	dir := t.TempDir()
	_, err := Download(context.Background(), nil, "https://x.example/rustdesk.deb", checksum("abc"), dir)
	if err == nil || !strings.Contains(err.Error(), "nil") {
		t.Fatalf("error = %v, want a nil-client error", err)
	}
	requireEmptyDir(t, dir)
}

func TestDownloadRequiresDestDir(t *testing.T) {
	srv := serveBody(t, "abc")

	t.Run("missing", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "absent")
		_, err := Download(context.Background(), srv.Client(), srv.URL+"/rustdesk.deb", checksum("abc"), dir)
		if err == nil || !strings.Contains(err.Error(), "destination") {
			t.Fatalf("error = %v, want an actionable destination error", err)
		}
	})

	t.Run("not a directory", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(dir, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Download(context.Background(), srv.Client(), srv.URL+"/rustdesk.deb", checksum("abc"), dir)
		if err == nil || !strings.Contains(err.Error(), "destination") {
			t.Fatalf("error = %v, want an actionable destination error", err)
		}
	})
}

func TestDownloadRespectsContextCancellation(t *testing.T) {
	srv := serveBody(t, "abc")
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Download(ctx, srv.Client(), srv.URL+"/rustdesk.deb", checksum("abc"), dir)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	requireEmptyDir(t, dir)
}

func TestDownloadRejectsBadChecksumFormat(t *testing.T) {
	srv := serveBody(t, "abc")
	bad := []string{"", "abc", strings.Repeat("z", 64), checksum("abc") + "0"}
	for _, sum := range bad {
		t.Run(sum, func(t *testing.T) {
			dir := t.TempDir()
			_, err := Download(context.Background(), srv.Client(), srv.URL+"/rustdesk.deb", sum, dir)
			if err == nil || !strings.Contains(err.Error(), "sha256") {
				t.Fatalf("error = %v, want a sha256 format error", err)
			}
			requireEmptyDir(t, dir)
		})
	}
}
