package plan

import (
	"strings"
	"testing"
)

// ADR-086. splitHeader walked the header as []rune, which turns each byte that
// is not valid UTF-8 into U+FFFD, so `@@ bad\xffname.txt 0 create` created
// bad�name.txt at exit 0 — another name than the one written. Every byte
// of a path and an anchor survives; a pattern with multibyte text splits as
// before. (A pattern holding an invalid byte is refused by regexp itself.)
func TestAHeaderKeepsEveryByteOfItsPath(t *testing.T) {
	hs, err := Parse(strings.NewReader("@@ bad\xffname.txt 0 create\nx\n@@ a.txt 1 replace anchor=\"x\xfey z\"\nA\n@@ b.txt /^fé (s)/ replace\nB\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(hs) != 3 {
		t.Fatalf("got %d hunks, want 3", len(hs))
	}
	if hs[0].Path != "bad\xffname.txt" {
		t.Errorf("path = %q, want %q", hs[0].Path, "bad\xffname.txt")
	}
	if hs[1].Anchor != "x\xfey z" {
		t.Errorf("anchor = %q, want %q", hs[1].Anchor, "x\xfey z")
	}
	if got := hs[2].Addr.StartPat.String(); got != "^fé (s)" {
		t.Errorf("pattern = %q, want %q", got, "^fé (s)")
	}
}
