// SPDX-License-Identifier: AGPL-3.0-or-later

// Package fetch downloads a release artifact and places it in its final
// location only after its SHA-256 checksum has been verified.
package fetch

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ErrChecksumMismatch reports that the downloaded artifact does not match the
// expected SHA-256 checksum. Callers match it with errors.Is.
var ErrChecksumMismatch = errors.New("fetch: checksum mismatch")

// maxArtifactBytes caps a single download. It is a variable rather than a
// constant only so tests can lower the limit instead of moving hundreds of
// megabytes through the test binary; production code never changes it.
var maxArtifactBytes int64 = 512 << 20

const (
	tempPrefix = ".partial-"
	fileMode   = 0o600
)

var sha256Pattern = regexp.MustCompile(`^[0-9A-Fa-f]{64}$`)

// Download fetches rawURL into destDir, verifies that the SHA-256 checksum of
// the body matches sha256hex (compared case-insensitively), and only then
// moves the verified file to destDir/filepath.Base(rawURL), returning that
// path.
//
// The body is streamed to a hidden .partial- file while it is hashed, so any
// file already at the final path is left untouched until the checksum has
// been verified. A failure removes the temporary file and returns an empty
// path. destDir must already exist; Download does not create it.
func Download(ctx context.Context, client *http.Client, rawURL, sha256hex, destDir string) (string, error) {
	if client == nil {
		return "", errors.New("fetch: HTTP client must not be nil")
	}
	if ctx == nil {
		return "", errors.New("fetch: context must not be nil")
	}
	if err := checkDestDir(destDir); err != nil {
		return "", err
	}
	name, err := finalName(rawURL)
	if err != nil {
		return "", err
	}
	if !sha256Pattern.MatchString(sha256hex) {
		return "", fmt.Errorf("fetch: sha256 %q must be 64 hex characters", sha256hex)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("fetch: build request for %s: %w", rawURL, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch: get %s: %w", rawURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch: get %s: unexpected status %s", rawURL, resp.Status)
	}
	if resp.ContentLength > maxArtifactBytes {
		return "", fmt.Errorf("fetch: %s declares %d bytes, larger than the %d-byte limit", rawURL, resp.ContentLength, maxArtifactBytes)
	}

	tmp, err := tempName(destDir)
	if err != nil {
		return "", err
	}
	// After a successful rename this is a no-op, so the defer covers every
	// failure path without needing a separate committed flag.
	defer func() { _ = os.Remove(tmp) }()

	actual, err := writeAndHash(tmp, resp.Body)
	if err != nil {
		return "", fmt.Errorf("fetch: download %s: %w", rawURL, err)
	}
	if !strings.EqualFold(actual, sha256hex) {
		return "", fmt.Errorf("%w: download of %s: expected %s, got %s",
			ErrChecksumMismatch, rawURL, strings.ToLower(sha256hex), actual)
	}

	final := filepath.Join(destDir, name)
	if err := os.Rename(tmp, final); err != nil {
		return "", fmt.Errorf("fetch: place %s: %w", final, err)
	}
	return final, nil
}

// checkDestDir verifies that dir already exists and is a directory.
func checkDestDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("fetch: destination directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("fetch: destination %s is not a directory", dir)
	}
	return nil
}

// finalName extracts the file name Download will use. A URL that does not end
// in a path element, or whose base is "", ".", "..", or "/", names no usable
// file: it is rejected instead of producing a hidden or traversal-prone name.
func finalName(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("fetch: URL %q is invalid: %w", rawURL, err)
	}
	name := filepath.Base(rawURL)
	if u.Path == "" || strings.HasSuffix(u.Path, "/") {
		return "", fmt.Errorf("fetch: URL %q does not end in a usable file name", rawURL)
	}
	switch name {
	case "", ".", "..", string(filepath.Separator):
		return "", fmt.Errorf("fetch: URL %q does not end in a usable file name", rawURL)
	}
	return name, nil
}

// tempName returns a fresh, unpredictable .partial- path inside dir.
func tempName(dir string) (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("fetch: generate temporary file name: %w", err)
	}
	return filepath.Join(dir, tempPrefix+hex.EncodeToString(buf[:])), nil
}

// writeAndHash streams r into path while hashing it and returns the lowercase
// hex SHA-256 of what was written. The file is synced before it is closed. A
// body larger than maxArtifactBytes fails instead of being truncated.
func writeAndHash(path string, r io.Reader) (string, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileMode)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(r, maxArtifactBytes+1))
	if err != nil {
		_ = f.Close()
		return "", err
	}
	if n > maxArtifactBytes {
		_ = f.Close()
		return "", fmt.Errorf("artifact is larger than the %d-byte limit", maxArtifactBytes)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
