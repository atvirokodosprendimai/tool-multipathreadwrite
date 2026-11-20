package read

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter"
)

// Review of #722. The working set cuts an entry into path and range with the
// read parser's own rule (addr.SpecColon, with this package's grammar as the
// validity test), so every spec main accepted splits as it did, and a colon
// inside a quoted key moves the cut only when nothing else makes the rest valid.
func TestPathCutsAnEntryWhereReadDoes(t *testing.T) {
	for _, tc := range []struct{ entry, path string }{
		{`doc.json:$["build:prod"]`, "doc.json"},
		{`doc.json:$.a["b:c"]`, "doc.json"},
		{`file:/x/, $.a/b,:$.c:d`, "file"}, // a spec main accepts, with a later :$. colon in it
		{"a:$.txt:2", "a:$.txt"},
		{"b:$[1].txt:1", "b:$[1].txt"},
		{"d:$.x/f.go:2", "d:$.x/f.go"},
		{`a:$["q:1`, `a:$["q`}, // a file name holding an unmatched quote
		{"plain.go:3-9", "plain.go"},
		{"bare.go", "bare.go"},
	} {
		if got := iter.Path(tc.entry); got != tc.path {
			t.Errorf("iter.Path(%q) = %q, want %q", tc.entry, got, tc.path)
		}
		if sp, err := ParseSpec(tc.entry); tc.entry != "bare.go" && (err != nil || sp.Path != tc.path) {
			t.Errorf("ParseSpec(%q) = %q, %v; want the path %q", tc.entry, sp.Path, err, tc.path)
		}
	}
}

// Review of #722, P1. With files `a` and `a:$["q`, the entry `a:$["q:1` is line 1
// of the second: the @N pointer and Remove must name it, not `a`.
func TestAFileNameHoldingAQuoteIsNotRedirectedByAnEntry(t *testing.T) {
	root := t.TempDir()
	const odd = `a:$["q`
	for _, name := range []string{"a", odd} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x\n"), 0o644); err != nil {
			t.Skipf("this filesystem keeps no file named %q: %v", name, err)
		}
	}
	var s iter.Set
	s.Add(odd + ":1")
	got, err := s.Resolve("@1")
	if err != nil || len(got) != 1 || iter.Path(got[0]) != odd {
		t.Fatalf("@1 resolves to %q (%v), whose path is %q; want %q", got, err, iter.Path(got[0]), odd)
	}
	if n := s.Remove("a"); n != 0 {
		t.Errorf("removing a removed %d entries of the file %s", n, odd)
	}
	if n := s.Remove(odd); n != 1 {
		t.Errorf("removing %s removed %d, want 1", odd, n)
	}
}

// Review of #722, P2-5. A quote inside a dot-style name is a plain character, so
// the range after it is kept, as on main.
func TestAQuoteInADotNameDoesNotSwallowTheNextRange(t *testing.T) {
	sp, err := ParseSpec(`.quality-harness.json:$.a"b,1`)
	if err != nil || len(sp.Ranges) != 2 || sp.Ranges[0].JSON == nil || sp.Ranges[1].Start != 1 || sp.Ranges[1].JSON != nil {
		t.Errorf("a quote in a dot name: %+v, %v; want the JSON path and line 1", sp.Ranges, err)
	}
}

// Final review of #722. A list that mixes ranges with a quoted key holding a
// colon or a comma stays refused, as on main: only ONE whole JSON path moves the
// cut, so no spec main accepted can be read differently.
func TestAListMixingRangesWithAQuotedKeyStaysRefused(t *testing.T) {
	for _, spec := range []string{`doc.json:1,$["a:b"]`, `doc.json:7-9, $["a,b"]`, `doc.json:$["a,b"],$["c:d"]`} {
		if sp, err := ParseSpec(spec); err == nil {
			t.Errorf("%s was accepted: %+v; main refuses it", spec, sp)
		}
	}
}

// Final review of #722, P1. `file:/x/, $.a/b,:$.c:d` is a spec for `file`, and an
// entry made of it must stay one: the @N pointer and Remove name `file`, not
// whatever follows its last `:$.`.
func TestAnEntryMainReadsAsFileIsNotRedirectedByALaterColon(t *testing.T) {
	const entry = `file:/x/, $.a/b,:$.c:d`
	if sp, err := ParseSpec(entry); err != nil || sp.Path != "file" {
		t.Fatalf("ParseSpec(%q) = %q, %v; main reads it as file", entry, sp.Path, err)
	}
	var s iter.Set
	s.Add(entry)
	got, err := s.Resolve("@1")
	if err != nil || len(got) != 1 || iter.Path(got[0]) != "file" {
		t.Fatalf("@1 resolves to %q (%v); its path must be file", got, err)
	}
	if n := s.Remove("file"); n != 1 {
		t.Errorf("removing file removed %d, want 1", n)
	}
}
