package plan

import (
	"strings"
	"testing"
)

// ADR-164 T6. An unknown op after a pattern address that a later field shows
// ended early names the cause: a peer's `/^\| `scripts/release.mjs` \| edit/`
// was refused `unknown op "\\|"`, because the unescaped / inside the pattern
// ends it. A misspelled op beside an option holding a / is not that case.
func TestAPatternEndedByASlashSaysSo(t *testing.T) {
	const hint = `ended at the first unescaped "/"`
	for _, c := range []struct {
		doc  string
		hint bool
	}{
		{"@@ t.md /^\\| `scripts/release.mjs` \\| edit/ replace\nZ\n", true},
		{"@@ t.md /x/ frob\nZ\n", false},
		{"@@ t.md /x/,+3 frob\nZ\n", false}, // a relative end is part of the address
		{"@@ t.md /x/ frob anchor=\"a/b\"\nZ\n", false},
		{"@@ t.md /x/ frob body=@dir/file\nZ\n", false},
	} {
		_, err := Parse(strings.NewReader(c.doc))
		if err == nil {
			t.Fatalf("%q parsed", c.doc)
		}
		msg := err.Error()
		if !strings.Contains(msg, "unknown op") {
			t.Errorf("%q: the old words are gone: %s", c.doc, msg)
		}
		if got := strings.Contains(msg, hint); got != c.hint {
			t.Errorf("%q: hint present %v, want %v: %s", c.doc, got, c.hint, msg)
		}
		if c.hint && !strings.Contains(msg, `\/`) {
			t.Errorf("%q: the hint does not say how to write the slash: %s", c.doc, msg)
		}
	}
	if _, err := Parse(strings.NewReader("@@ t.md /^\\| `scripts\\/release.mjs` \\| edit/ replace\nZ\n")); err != nil {
		t.Errorf("the escaped pattern does not parse: %v", err)
	}
}
