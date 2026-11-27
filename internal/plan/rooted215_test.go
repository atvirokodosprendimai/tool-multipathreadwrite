package plan

import (
	"strings"
	"testing"
)

// ADR-215 Decision 4. A header whose first token begins with "/" and has no
// second "/" is still a document error (exit 2), and its sentence now says what
// the caller did. Nothing about how the splitter reads a token changes: every
// other spelling that starts with "/" parses as it did.
func TestAHeaderWhoseFirstTokenBeginsWithASlashSaysSo(t *testing.T) {
	const says = "a plan path is relative to the root and never begins with /"
	for _, plan := range []string{"@@ /zz.txt 1 replace\nX\n", "@@ /dd 1 replace\nX\n", "@@ /zz.txt 0 create\nX\n"} {
		_, err := Parse(strings.NewReader(plan))
		if err == nil {
			t.Fatalf("%q parsed", plan)
		}
		if !strings.Contains(err.Error(), "unterminated pattern in header: expected a second /") || !strings.Contains(err.Error(), says) {
			t.Errorf("%q: %v, want the unterminated-pattern error and %q", plan, err, says)
		}
	}

	// An unterminated pattern that is not the first token keeps the old words.
	if _, err := Parse(strings.NewReader("@@ f.txt /unclosed replace\nX\n")); err == nil || strings.Contains(err.Error(), says) {
		t.Errorf("a later unterminated pattern: %v, want an error without %q", err, says)
	}

	// What must not change.
	for _, c := range []struct {
		plan, path, addr, anchor string
		body                     []string
	}{
		{"@@ /a b/c.txt 1 replace\nX\n", "/a b/c.txt", "1", "", []string{"X"}},
		{"@@ /a\"b\"/c.txt 1 replace\nX\n", "/a\"b\"/c.txt", "1", "", []string{"X"}},
		{"@@ \"/a\"b/c.txt 1 replace\nX\n", "/ab/c.txt", "1", "", []string{"X"}},
		{"@@ dir/x /a\\/b/ replace anchor=\"x/y\"\nX\n", "dir/x", "/a\\/b/", "x/y", []string{"X"}},
		{"@@ dir/x /a/,/b/ replace\nX\n", "dir/x", "/a/,/b/", "", []string{"X"}},
		{"@@ @1 /re/ replace\nX\n", "@1", "/re/", "", []string{"X"}},
		{"@@ dir/x /re/ replace anchor=prefix /a  b/\nX\n", "dir/x", "/re/", "prefix /a  b/", []string{"X"}},
		{"@@ out.txt 0 create body=1\n@@ /zz.txt 1 replace\n", "out.txt", "0", "", []string{"@@ /zz.txt 1 replace"}},
	} {
		h, err := Parse(strings.NewReader(c.plan))
		if err != nil || len(h) != 1 {
			t.Errorf("%q: %v, %d hunk(s), want one hunk", c.plan, err, len(h))
			continue
		}
		if h[0].Path != c.path || h[0].Addr.String() != c.addr || h[0].Anchor != c.anchor || strings.Join(h[0].Body, "\n") != strings.Join(c.body, "\n") {
			t.Errorf("%q: path %q addr %q anchor %q body %q, want %q %q %q %q", c.plan, h[0].Path, h[0].Addr, h[0].Anchor, h[0].Body, c.path, c.addr, c.anchor, c.body)
		}
	}
}
