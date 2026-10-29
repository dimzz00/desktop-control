// SPDX-License-Identifier: AGPL-3.0-or-later

// Package deps parses deps.lock, the file that pins the third-party
// dependencies kendali installs and verifies.
//
// The format is deliberately tiny and line based:
//
//	name version url sha256
//
// Fields are separated by spaces or tabs. Blank lines and lines whose first
// non-space character is '#' are ignored; comments cannot trail a data line.
// Every data line must carry exactly four fields and every entry must pass
// Validate, so a malformed lock file is rejected instead of being guessed at.
package deps

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// maxLineBytes bounds a single line. Real entries are far shorter; the limit
// only exists so a hostile or corrupt file fails with a clear error instead of
// unbounded memory use.
const maxLineBytes = 1 << 20

var (
	namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	shaPattern  = regexp.MustCompile(`^[0-9A-Fa-f]{64}$`)
)

// Entry is one pinned dependency from deps.lock.
type Entry struct {
	Name    string
	Version string
	URL     string
	SHA256  string
}

// Validate reports whether e is safe to record and install. It names the
// offending field in every error.
func (e Entry) Validate() error {
	if !namePattern.MatchString(e.Name) {
		return fmt.Errorf("name %q must match %s", e.Name, namePattern)
	}
	if e.Version == "" {
		return errors.New("version must not be empty")
	}
	u, err := url.Parse(e.URL)
	if err != nil {
		return fmt.Errorf("url %q is not a valid URL: %v", e.URL, err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("url %q must use https", e.URL)
	}
	if u.Host == "" {
		return fmt.Errorf("url %q must name a host", e.URL)
	}
	if !shaPattern.MatchString(e.SHA256) {
		return fmt.Errorf("sha256 %q must be exactly 64 hex characters", e.SHA256)
	}
	return nil
}

// Load opens the deps.lock at path and parses it. A missing file is reported
// as an error wrapping os.ErrNotExist.
func Load(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("deps: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return Parse(f)
}

// Parse reads deps.lock from r. A file with no data lines yields an empty,
// non-nil slice. Any read error from r is returned wrapped, and the first bad
// line aborts the parse.
func Parse(r io.Reader) ([]Entry, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 4096), maxLineBytes)

	entries := make([]Entry, 0, 4)
	firstLine := make(map[string]int)

	for line := 1; sc.Scan(); line++ {
		text := sc.Text()
		trimmed := strings.TrimSpace(text)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		fields := splitFields(text)
		if len(fields) != 4 {
			return nil, fmt.Errorf("deps: line %d: expected 4 fields, got %d", line, len(fields))
		}

		e := Entry{Name: fields[0], Version: fields[1], URL: fields[2], SHA256: fields[3]}
		if err := e.Validate(); err != nil {
			return nil, fmt.Errorf("deps: line %d: %w", line, err)
		}
		if first, ok := firstLine[e.Name]; ok {
			return nil, fmt.Errorf("deps: line %d: duplicate name %q (first defined on line %d)", line, e.Name, first)
		}
		firstLine[e.Name] = line
		entries = append(entries, e)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("deps: read: %w", err)
	}
	return entries, nil
}

// splitFields splits on the two separators the format allows. Unlike
// strings.Fields it does not treat other Unicode whitespace as a separator, so
// such bytes end up inside a field and are rejected by Validate.
func splitFields(line string) []string {
	return strings.FieldsFunc(line, func(r rune) bool {
		return r == ' ' || r == '\t'
	})
}
