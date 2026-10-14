package read

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-137. The width is counted in characters: a line of thirty two-byte
// characters is sixty bytes and fits in a width of forty.
func TestTheWidthIsCountedInCharactersNotBytes(t *testing.T) {
	root := t.TempDir()
	line := strings.Repeat("é", 30)
	if err := os.WriteFile(filepath.Join(root, "u.txt"), []byte(line+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sp, err := ParseSpec("u.txt")
	if err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	observed, _ := Run(&sb, root, []Spec{sp}, Options{Numbers: true, MaxCols: 40})
	if strings.Contains(sb.String(), "[cols ") || !strings.Contains(sb.String(), line) {
		t.Errorf("a 30-character line was cut by a width of 40:\n%s", sb.String())
	}
	if observed["u.txt"].Spans != nil {
		t.Errorf("a line served whole was recorded as %v, want the whole-file observation", observed["u.txt"].Spans)
	}
}
