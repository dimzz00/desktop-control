// SPDX-License-Identifier: AGPL-3.0-or-later

package deps

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
)

// sha256Hex is a valid checksum: 64 lowercase hex characters.
const sha256Hex = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestParse(t *testing.T) {
	in := "# komentar\n" +
		"\n" +
		"rustdesk 1.4.2 https://x.example/rustdesk.deb " + sha256Hex + "\n" +
		"   \t\n" +
		"  # komentar menjorok\n" +
		"tailscale\t1.66.0\thttps://pkgs.tailscale.com/x.deb\t" + strings.ToUpper(sha256Hex)

	es, err := Parse(strings.NewReader(in))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := []Entry{
		{Name: "rustdesk", Version: "1.4.2", URL: "https://x.example/rustdesk.deb", SHA256: sha256Hex},
		{Name: "tailscale", Version: "1.66.0", URL: "https://pkgs.tailscale.com/x.deb", SHA256: strings.ToUpper(sha256Hex)},
	}
	if len(es) != len(want) {
		t.Fatalf("got %d entries, want %d: %v", len(es), len(want), es)
	}
	for i := range want {
		if es[i] != want[i] {
			t.Fatalf("entry %d = %+v, want %+v", i, es[i], want[i])
		}
	}
}

func TestParseEmpty(t *testing.T) {
	for _, in := range []string{"", "\n\n", "# only a comment\n", "  \t \n# komentar\n\n"} {
		t.Run(strings.ReplaceAll(in, "\n", `\n`), func(t *testing.T) {
			es, err := Parse(strings.NewReader(in))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if es == nil {
				t.Fatal("Parse must return an empty slice, not nil")
			}
			if len(es) != 0 {
				t.Fatalf("got %d entries, want 0: %v", len(es), es)
			}
		})
	}
}

func TestParseRejectsBadLine(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "too few fields",
			in:   "# komentar\nrustdesk 1.4.2 https://x.example/y.deb\n",
			want: []string{"line 2", "expected 4 fields, got 3"},
		},
		{
			name: "only one field",
			in:   "rustdesk\n",
			want: []string{"line 1", "expected 4 fields, got 1"},
		},
		{
			name: "too many fields",
			in:   "rustdesk 1.4.2 https://x.example/y.deb " + sha256Hex + " extra\n",
			want: []string{"line 1", "expected 4 fields, got 5"},
		},
		{
			name: "trailing comment",
			in:   "rustdesk 1.4.2 https://x.example/y.deb " + sha256Hex + " # catatan\n",
			want: []string{"expected 4 fields, got 6"},
		},
		{
			name: "uppercase name",
			in:   "RustDesk 1.4.2 https://x.example/y.deb " + sha256Hex + "\n",
			want: []string{"line 1", `name "RustDesk"`},
		},
		{
			name: "name starting with dash",
			in:   "-rustdesk 1.4.2 https://x.example/y.deb " + sha256Hex + "\n",
			want: []string{"name"},
		},
		{
			name: "name with slash",
			in:   "team/rustdesk 1.4.2 https://x.example/y.deb " + sha256Hex + "\n",
			want: []string{"name"},
		},
		{
			name: "http url",
			in:   "rustdesk 1.4.2 http://x.example/y.deb " + sha256Hex + "\n",
			want: []string{"line 1", "https"},
		},
		{
			name: "ftp url",
			in:   "rustdesk 1.4.2 ftp://x.example/y.deb " + sha256Hex + "\n",
			want: []string{"https"},
		},
		{
			name: "relative url",
			in:   "rustdesk 1.4.2 x.example/y.deb " + sha256Hex + "\n",
			want: []string{"https"},
		},
		{
			name: "https without host",
			in:   "rustdesk 1.4.2 https: " + sha256Hex + "\n",
			want: []string{"host"},
		},
		{
			name: "short checksum",
			in:   "rustdesk 1.4.2 https://x.example/y.deb " + sha256Hex[:63] + "\n",
			want: []string{"sha256", "64 hex"},
		},
		{
			name: "long checksum",
			in:   "rustdesk 1.4.2 https://x.example/y.deb " + sha256Hex + "a\n",
			want: []string{"sha256", "64 hex"},
		},
		{
			name: "non-hex checksum",
			in:   "rustdesk 1.4.2 https://x.example/y.deb " + strings.Repeat("g", 64) + "\n",
			want: []string{"sha256"},
		},
		{
			name: "duplicate name",
			in:   "rustdesk 1.4.2 https://x.example/a.deb " + sha256Hex + "\nrustdesk 1.4.3 https://x.example/b.deb " + sha256Hex + "\n",
			want: []string{"line 2", `duplicate name "rustdesk"`, "line 1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es, err := Parse(strings.NewReader(tt.in))
			if err == nil {
				t.Fatalf("want error, got entries %v", es)
			}
			if es != nil {
				t.Fatalf("on error Parse must return entries as nil, got %v", es)
			}
			for _, want := range tt.want {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error %q must contain %q", err, want)
				}
			}
		})
	}
}

func TestParsePropagatesReaderError(t *testing.T) {
	boom := errors.New("boom")
	es, err := Parse(iotest.ErrReader(boom))
	if !errors.Is(err, boom) {
		t.Fatalf("Parse error = %v, want it to wrap %v", err, boom)
	}
	if es != nil {
		t.Fatalf("entries = %v, want nil", es)
	}
}

func TestEntryValidate(t *testing.T) {
	valid := Entry{
		Name:    "rustdesk",
		Version: "1.4.2",
		URL:     "https://github.com/rustdesk/rustdesk/releases/download/1.4.2/rustdesk.deb",
		SHA256:  sha256Hex,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid entry rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Entry)
		want   string
	}{
		{"empty name", func(e *Entry) { e.Name = "" }, "name"},
		{"uppercase name", func(e *Entry) { e.Name = "RustDesk" }, "name"},
		{"dot start", func(e *Entry) { e.Name = ".rustdesk" }, "name"},
		{"space in name", func(e *Entry) { e.Name = "rust desk" }, "name"},
		{"name with plus", func(e *Entry) { e.Name = "rust+desk" }, "name"},
		{"empty version", func(e *Entry) { e.Version = "" }, "version"},
		{"empty url", func(e *Entry) { e.URL = "" }, "url"},
		{"unparseable url", func(e *Entry) { e.URL = "https://x.example/\x7f" }, "url"},
		{"http url", func(e *Entry) { e.URL = "http://x.example/y.deb" }, "https"},
		{"file url", func(e *Entry) { e.URL = "file:///y.deb" }, "https"},
		{"relative url", func(e *Entry) { e.URL = "y.deb" }, "https"},
		{"https no host", func(e *Entry) { e.URL = "https:///y.deb" }, "host"},
		{"empty checksum", func(e *Entry) { e.SHA256 = "" }, "sha256"},
		{"short checksum", func(e *Entry) { e.SHA256 = sha256Hex[:63] }, "sha256"},
		{"long checksum", func(e *Entry) { e.SHA256 = sha256Hex + "0" }, "sha256"},
		{"non-hex checksum", func(e *Entry) { e.SHA256 = strings.Repeat("z", 64) }, "sha256"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := valid
			tt.mutate(&e)
			err := e.Validate()
			if err == nil {
				t.Fatal("want error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error %q must name %q", err, tt.want)
			}
		})
	}
}

func TestEntryValidateAcceptsUppercaseChecksum(t *testing.T) {
	e := Entry{Name: "rustdesk", Version: "1.4.2", URL: "https://x.example/y.deb", SHA256: strings.ToUpper(sha256Hex)}
	if err := e.Validate(); err != nil {
		t.Fatalf("uppercase checksum must be accepted: %v", err)
	}
}

func TestLoad(t *testing.T) {
	p := filepath.Join(t.TempDir(), "deps.lock")
	body := "# deps.lock\n" +
		"rustdesk 1.4.2 https://x.example/rustdesk.deb " + sha256Hex + "\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	es, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(es) != 1 || es[0].Name != "rustdesk" || es[0].SHA256 != sha256Hex {
		t.Fatalf("Load = %v", es)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "absent.lock")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Load error = %v, want os.ErrNotExist", err)
	}
}

func TestLoadReportsLineNumber(t *testing.T) {
	p := filepath.Join(t.TempDir(), "deps.lock")
	body := "# deps.lock\n\nrustdesk 1.4.2 https://x.example/rustdesk.deb\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err == nil || !strings.Contains(err.Error(), "line 3") {
		t.Fatalf("Load error = %v, want line 3", err)
	}
}

func FuzzParse(f *testing.F) {
	f.Add("")
	f.Add("# komentar\nrustdesk 1.4.2 https://x.example/y.deb " + sha256Hex + "\n")
	f.Add("rustdesk 1.4.2 http://x.example/y.deb " + strings.Repeat("a", 64) + "\n")
	f.Add("rustdesk 1.4.2 https://x.example/y.deb")

	f.Fuzz(func(t *testing.T, in string) {
		es, err := Parse(strings.NewReader(in))
		if err != nil {
			return
		}
		seen := make(map[string]bool, len(es))
		for _, e := range es {
			if err := e.Validate(); err != nil {
				t.Fatalf("Parse returned an invalid entry %+v: %v", e, err)
			}
			if seen[e.Name] {
				t.Fatalf("Parse returned duplicate name %q", e.Name)
			}
			seen[e.Name] = true
		}
	})
}
