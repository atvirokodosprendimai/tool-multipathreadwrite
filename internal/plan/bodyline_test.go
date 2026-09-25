package plan

import (
	"strings"
	"testing"
)

// ADR-070 T2. Blind reading 04 wrote body= on a line of its own under the
// header, as body=1 and as body=status: x. With no count on the header that
// line was body, and mrw wrote it into the file. It is refused now; counting
// the body is how a file that really starts with such a line is written.
func TestABodyLineThatIsABodyCountIsRefused(t *testing.T) {
	for _, first := range []string{"body=1", "body=@b.txt", "body=", "body=status: x", "  body=2"} {
		_, err := Parse(strings.NewReader("@@ f 1 replace\n" + first + "\nX\n"))
		if err == nil || !strings.Contains(err.Error(), "belongs ON the header") {
			t.Errorf("first body line %q: err = %v, want a refusal saying body= belongs ON the header", first, err)
		}
	}
	for _, doc := range []string{
		"@@ f 1 replace\nX\nbody=1\n",   // not the first body line
		"@@ f 1 replace\nx body=1\n",    // not at the start of the line
		"@@ a.txt - rename\nbody=1\n",   // a rename's body is a path
		"@@ f 1 replace\nbodyguard=1\n", // not the option
	} {
		if _, err := Parse(strings.NewReader(doc)); err != nil {
			t.Errorf("%q was refused: %v", doc, err)
		}
	}
}

func TestACountedBodyWritesALiteralBodyCountLine(t *testing.T) {
	hs, err := Parse(strings.NewReader("@@ f 1 replace body=1\nbody=1\n"))
	if err != nil {
		t.Fatalf("a counted body starting with body=1 was refused: %v", err)
	}
	if len(hs) != 1 || len(hs[0].Body) != 1 || hs[0].Body[0] != "body=1" {
		t.Errorf("hunks = %+v, want one body line \"body=1\"", hs)
	}
}
