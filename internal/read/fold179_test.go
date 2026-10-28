package read

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const fold179Body = "head\nx 1\nx 22\nx 3\nmid\nsame\nsame\nsame\n\n\ny 1\nother\ny 2\n"

// ADR-179 Decision (c). A run of two or more non-blank lines identical, or
// identical but for digits, is its first line and one `-- ×N: lines A-B like A`
// line (with the digits note when they are not identical); a run broken by one
// other line is two lines, blank lines never fold, and only the first line of a
// run is recorded as read.
func TestFoldCollapsesRepeatedAndDigitOnlyDifferentLines(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte(fold179Body), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	observed, problems := Run(&out, root, []Spec{{Path: "f.txt", Raw: "f.txt"}}, Options{Numbers: true, Fold: true})
	text := out.String()
	for _, w := range []string{
		"    1| head\n    2| x 1\n-- ×3: lines 2-4 like 2, differing only in digits\n    5| mid\n    6| same\n-- ×3: lines 6-8 like 6\n    9| \n   10| \n   11| y 1\n   12| other\n   13| y 2\n",
	} {
		if !strings.Contains(text, w) {
			t.Errorf("the fold printed\n%s\nwant it to hold\n%s", text, w)
		}
	}
	if strings.Contains(text, "x 22") || strings.Contains(text, "x 3\n") {
		t.Errorf("a folded line was printed:\n%s", text)
	}
	if got := observed["f.txt"].Spans; problems != 0 || !reflect.DeepEqual(got, [][2]int{{1, 2}, {5, 6}, {9, 13}}) {
		t.Errorf("observed %v (problems %d), want only the lines shown: [[1 2] [5 6] [9 13]]", got, problems)
	}
	out.Reset()
	Run(&out, root, []Spec{{Path: "f.txt", Raw: "f.txt"}}, Options{Numbers: true})
	if strings.Contains(out.String(), "×") || !strings.Contains(out.String(), "x 22") {
		t.Errorf("a read without Fold folded:\n%s", out.String())
	}
}
