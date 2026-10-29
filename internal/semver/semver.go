// SPDX-License-Identifier: AGPL-3.0-or-later

// Package semver parses and compares the dotted version numbers kendali reads
// from VERSION and from dependency pins.
//
// The grammar is a deliberately small subset of Semantic Versioning 2.0.0:
// three dot-separated numeric components and an optional prerelease suffix:
//
//	[v]MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]
//
// A leading "v" is accepted so tags such as "v1.4.2" parse directly. Build
// metadata after "+" is validated only for position: it is ignored, so
// "1.4.2+build.5" and "1.4.2" yield the same Version.
package semver

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Version is a parsed semantic version.
type Version struct {
	// Major is the major version component.
	Major int
	// Minor is the minor version component.
	Minor int
	// Patch is the patch version component.
	Patch int
	// Pre is the raw prerelease suffix without its leading "-", or "" when the
	// version carries no prerelease.
	Pre string
}

// Parse parses s as a semantic version, accepting an optional leading "v" and
// ignoring any "+build" metadata that follows the version. It rejects empty
// input, anything other than exactly three numeric components, components with
// a sign or non-digit byte, components that overflow int, and an empty or
// whitespace-bearing prerelease. Leading zeros in a numeric component are
// accepted; String normalizes them away.
func Parse(s string) (Version, error) {
	orig := s
	fail := func(format string, args ...any) (Version, error) {
		return Version{}, fmt.Errorf("semver: %q: "+format, append([]any{orig}, args...)...)
	}

	if s == "" {
		return fail("empty version")
	}

	// Build metadata is ignored, but only once the version part is known to be
	// present: "+build.5" alone leaves an empty version and is rejected below.
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimPrefix(s, "v")

	var pre string
	if i := strings.IndexByte(s, '-'); i >= 0 {
		s, pre = s[:i], s[i+1:]
		if pre == "" {
			return fail("empty prerelease")
		}
		if strings.IndexFunc(pre, unicode.IsSpace) >= 0 {
			return fail("prerelease %q must not contain whitespace", pre)
		}
	}

	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return fail("want MAJOR.MINOR.PATCH, got %d component(s)", len(parts))
	}

	var nums [3]int
	for i, part := range parts {
		n, err := parseComponent(part)
		if err != nil {
			return fail("component %d: %v", i+1, err)
		}
		nums[i] = n
	}

	return Version{Major: nums[0], Minor: nums[1], Patch: nums[2], Pre: pre}, nil
}

// parseComponent converts one numeric version component. Requiring digits
// before calling strconv.Atoi keeps "+2" and "-2" from being accepted, and Atoi
// itself rejects values that do not fit in int.
func parseComponent(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty component")
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, fmt.Errorf("%q is not a non-negative decimal integer", s)
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("%q does not fit in int", s)
	}
	return n, nil
}

// Compare orders v against o and returns -1, 0, or 1 when v is lower, equal, or
// higher. Numeric components dominate; a version without a prerelease outranks
// the same numeric version with one ("1.4.2" > "1.4.2-rc.1"). Prerelease
// strings are compared lexicographically byte-by-byte, not by the SemVer §11
// identifier rules, so "1.4.2-rc.10" sorts below "1.4.2-rc.2" here.
func (v Version) Compare(o Version) int {
	if c := compareInt(v.Major, o.Major); c != 0 {
		return c
	}
	if c := compareInt(v.Minor, o.Minor); c != 0 {
		return c
	}
	if c := compareInt(v.Patch, o.Patch); c != 0 {
		return c
	}
	switch {
	case v.Pre == o.Pre:
		return 0
	case v.Pre == "":
		return 1
	case o.Pre == "":
		return -1
	default:
		return strings.Compare(v.Pre, o.Pre)
	}
}

// compareInt returns -1, 0, or 1 for a < b, a == b, and a > b.
func compareInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// String formats v as MAJOR.MINOR.PATCH, followed by "-" and the prerelease
// when one is set. Build metadata is never retained, so the result always
// parses back to an equal Version.
func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Pre != "" {
		s += "-" + v.Pre
	}
	return s
}
