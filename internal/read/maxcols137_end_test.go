package read

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The Codex review of #367. A /start/,/end/ range may end on the long line; the
// window is centred on the end pattern's match, not shown from column 1.
func TestTheWindowCentresOnTheEndOfAStartEndRange(t *testing.T) {
	root := t.TempDir()
	body := "START\n" + strings.Repeat("x", 300) + "ENDMARK" + strings.Repeat("y", 93) + "\n"
	if err := os.WriteFile(filepath.Join(root, "r.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	sp, err := ParseSpec("r.txt:/START/,/ENDMARK/")
	if err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	Run(&sb, root, []Spec{sp}, Options{Numbers: true, MaxCols: 40})
	if !strings.Contains(sb.String(), "ENDMARK") {
		t.Errorf("the window on the ending line hides the end pattern:\n%s", sb.String())
	}
}

// The window is found by byte offsets in the original line, so a window of a
// few characters costs nothing in the size of the line, and it still cuts on
// characters: the offset helper is exact on multibyte text and past the end.
func TestRuneOffsetIsExactOnMultibyteText(t *testing.T) {
	s := "aé世b"
	for n, want := range []int{0, 1, 3, 6, 7} {
		if got := runeOffset(s, n); got != want {
			t.Errorf("runeOffset(%q, %d) = %d, want %d", s, n, got, want)
		}
	}
	if got := runeOffset(s, 99); got != len(s) {
		t.Errorf("runeOffset past the end = %d, want %d", got, len(s))
	}
}
