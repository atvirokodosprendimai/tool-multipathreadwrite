package read

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-073 T2. A read of a UTF-16 file served byte-split lines with nothing
// saying why they look wrong. It is served as before — the read did its job —
// with one note naming the encoding and that a write to it is refused.
func TestAForeignFileIsServedWithANote(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "u.txt"), []byte("\xff\xfea\x00\n\x00"), 0o644); err != nil {
		t.Fatal(err)
	}
	sp, err := ParseSpec("u.txt")
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	observed, problems := Run(&b, root, []Spec{sp}, Options{Numbers: true})
	if problems != 0 || len(observed) != 1 {
		t.Fatalf("an encoded file was not served: problems %d, observed %v\n%q", problems, observed, b.String())
	}
	if !strings.Contains(b.String(), "-- note: u.txt is UTF-16 (BOM FF FE): served as bytes; a write to it is refused") {
		t.Fatalf("no note names the encoding:\n%q", b.String())
	}
}

// The pair: a UTF-8 file carries no note.
func TestAUTF8FileGetsNoNote(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("plain\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sp, err := ParseSpec("a.txt")
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	Run(&b, root, []Spec{sp}, Options{Numbers: true})
	if strings.Contains(b.String(), "-- note:") {
		t.Fatalf("a UTF-8 file got a note:\n%q", b.String())
	}
}
