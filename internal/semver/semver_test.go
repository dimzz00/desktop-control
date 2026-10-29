// SPDX-License-Identifier: AGPL-3.0-or-later

package semver

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in   string
		want Version
	}{
		{"1.4.2", Version{Major: 1, Minor: 4, Patch: 2}},
		{"v1.4.2", Version{Major: 1, Minor: 4, Patch: 2}},
		{"1.4.2-rc.1", Version{Major: 1, Minor: 4, Patch: 2, Pre: "rc.1"}},
		{"v1.4.2-beta", Version{Major: 1, Minor: 4, Patch: 2, Pre: "beta"}},
		{"1.4.2+build.5", Version{Major: 1, Minor: 4, Patch: 2}},
		{"v1.4.2-rc.1+build.5", Version{Major: 1, Minor: 4, Patch: 2, Pre: "rc.1"}},
		{"0.0.0", Version{}},
		{"10.20.30", Version{Major: 10, Minor: 20, Patch: 30}},
	}

	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := Parse(c.in)
			if err != nil {
				t.Fatalf("Parse(%q): %v", c.in, err)
			}
			if got != c.want {
				t.Fatalf("Parse(%q) = %+v, want %+v", c.in, got, c.want)
			}
		})
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"only v", "v"},
		{"missing patch", "1.4"},
		{"missing minor and patch", "1"},
		{"too many components", "1.4.2.3"},
		{"non-numeric minor", "1.x"},
		{"non-numeric patch", "1.4.x"},
		{"negative patch", "1.4.-2"},
		{"leading plus component", "1.4.+2"},
		{"leading zeros are digits but sign is not", "-1.4.2"},
		{"overflow", "1.4.99999999999999999999999999"},
		{"empty prerelease", "1.4.2-"},
		{"uppercase V prefix", "V1.4.2"},
		{"leading space", " 1.4.2"},
		{"trailing space", "1.4.2 "},
		{"inner space", "1. 4.2"},
		{"space in prerelease", "1.4.2-rc 1"},
		{"dot only", "."},
		{"two dots", ".."},
		{"build metadata only", "+build.5"},
		{"non-numeric major", "x.4.2"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Parse(c.in)
			if err == nil {
				t.Fatalf("Parse(%q) = %+v, want error", c.in, got)
			}
			if got != (Version{}) {
				t.Fatalf("Parse(%q) returned %+v alongside error, want zero Version", c.in, got)
			}
		})
	}
}

func TestParseErrorNamesInput(t *testing.T) {
	_, err := Parse("1.x")
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), `"1.x"`) {
		t.Fatalf("error %q must quote the rejected input", err)
	}
}

func TestParseBoundaries(t *testing.T) {
	maxInt := strconv.Itoa(math.MaxInt)
	got, err := Parse(maxInt + "." + maxInt + "." + maxInt)
	if err != nil {
		t.Fatalf("Parse(max int) : %v", err)
	}
	want := Version{Major: math.MaxInt, Minor: math.MaxInt, Patch: math.MaxInt}
	if got != want {
		t.Fatalf("Parse(max int) = %+v, want %+v", got, want)
	}

	if _, err := Parse("1.4." + maxInt + "0"); err == nil {
		t.Fatal("component beyond int range must be rejected")
	}
}

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.4.2", "1.4.2", 0},
		{"1.4.3", "1.4.2", 1},
		{"1.4.2", "1.5.0", -1},
		{"1.4.2-rc.1", "1.4.2", -1},
		{"v1.4.2", "1.4.1", 1},
		{"2.0.0", "1.99.99", 1},
		{"1.4.2-rc.1", "1.4.2-rc.2", -1},
		{"1.4.2-beta", "1.4.2-alpha", 1},
		{"1.4.2-alpha", "1.4.3-alpha", -1},
		{"0.0.0", "0.0.1", -1},
	}

	for _, c := range cases {
		t.Run(c.a+"_"+c.b, func(t *testing.T) {
			a, err := Parse(c.a)
			if err != nil {
				t.Fatalf("Parse(%q): %v", c.a, err)
			}
			b, err := Parse(c.b)
			if err != nil {
				t.Fatalf("Parse(%q): %v", c.b, err)
			}
			if got := a.Compare(b); got != c.want {
				t.Fatalf("%q.Compare(%q) = %d, want %d", c.a, c.b, got, c.want)
			}
			if got := b.Compare(a); got != -c.want {
				t.Fatalf("%q.Compare(%q) = %d, want %d (antisymmetry)", c.b, c.a, got, -c.want)
			}
		})
	}
}

func TestComparePrereleaseIsLexicographic(t *testing.T) {
	// SemVer §11 would rank rc.10 above rc.2 numerically; this package
	// deliberately compares the raw prerelease string lexicographically, so
	// "rc.10" < "rc.2". Pinning that choice keeps it from changing silently.
	a, _ := Parse("1.4.2-rc.10")
	b, _ := Parse("1.4.2-rc.2")
	if got := a.Compare(b); got != -1 {
		t.Fatalf("lexicographic prerelease compare = %d, want -1", got)
	}
}

func TestStringRoundTrip(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"1.4.2", "1.4.2"},
		{"v1.4.2", "1.4.2"},
		{"1.4.2-rc.1", "1.4.2-rc.1"},
		{"v1.4.2-beta", "1.4.2-beta"},
		{"1.4.2+build.5", "1.4.2"},
		{"v1.4.2-rc.1+build.5", "1.4.2-rc.1"},
		{"0.0.0", "0.0.0"},
	}

	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			v, err := Parse(c.in)
			if err != nil {
				t.Fatalf("Parse(%q): %v", c.in, err)
			}
			if got := v.String(); got != c.want {
				t.Fatalf("Parse(%q).String() = %q, want %q", c.in, got, c.want)
			}
			again, err := Parse(v.String())
			if err != nil {
				t.Fatalf("re-Parse(%q): %v", v.String(), err)
			}
			if again != v {
				t.Fatalf("round trip changed version: %+v -> %+v", v, again)
			}
		})
	}
}

func TestStringDirect(t *testing.T) {
	v := Version{Major: 1, Minor: 4, Patch: 2, Pre: "rc.1"}
	if got := v.String(); got != "1.4.2-rc.1" {
		t.Fatalf("String() = %q", got)
	}
}

func FuzzParseRoundTrip(f *testing.F) {
	for _, seed := range []string{"", "v", "1.4.2", "v1.4.2-rc.1", "1.4.2+build.5", "1.x", "1.4.2-"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		v, err := Parse(in)
		if err != nil {
			return
		}
		again, err := Parse(v.String())
		if err != nil {
			t.Fatalf("Parse(%q) succeeded but Parse(%q) failed: %v", in, v.String(), err)
		}
		if again != v {
			t.Fatalf("round trip %q -> %+v -> %+v", in, v, again)
		}
	})
}
