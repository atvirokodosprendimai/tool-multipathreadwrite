package apply

import (
	"strings"
	"testing"
)

// ADR-164 T1. A multi-line replace refused for having no anchor= says, for
// each of sha= and lines= it carries, that the guard does not stand in: a field
// report's caller gave a matching sha= and took it for the guard, and lines=
// can be taken for it the same way. Without them the refusal is as before, so a
// clause is evidence of its guard, not noise on every refusal.
func TestAShaGuardDoesNotStandInForAnAnchor(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", "1\n2\n3\n4\n5\n6\n7\n")
	sha := shaOfFile(t, root, "f.txt")
	const shaClause = "sha= checks that the whole file is unchanged, not which lines this range covers"
	const linesClause = "lines= checks how many lines this range covers, not which"

	for _, tc := range []struct {
		name      string
		sha       string
		lines     int
		wantSHA   bool
		wantLines bool
	}{
		{"with sha=", sha, -1, true, false},
		{"with lines=", "", 3, false, true},
		{"with both", sha, 3, true, true},
		{"with neither", "", -1, false, false},
	} {
		res, err := Apply(root, []Input{
			{Path: "f.txt", Start: 3, End: 5, Op: "replace", Body: []string{"NEW"}, SHA: tc.sha, Lines: tc.lines, Index: 0},
		}, Options{})
		if err != nil {
			t.Fatal(err)
		}
		if res.Failed != 1 {
			t.Fatalf("%s: an anchorless multi-line replace was not refused: %+v", tc.name, res.Hunks)
		}
		got := res.Hunks[0].Reason
		if !strings.Contains(got, "carries no anchor=") || !strings.Contains(got, `say anchor="<text from the first line>"`) {
			t.Errorf("%s: the refusal lost its words: %s", tc.name, got)
		}
		if has := strings.Contains(got, shaClause); has != tc.wantSHA {
			t.Errorf("%s: sha= clause present %v, want %v: %s", tc.name, has, tc.wantSHA, got)
		}
		if has := strings.Contains(got, linesClause); has != tc.wantLines {
			t.Errorf("%s: lines= clause present %v, want %v: %s", tc.name, has, tc.wantLines, got)
		}
		if got, want := read(t, root, "f.txt"), "1\n2\n3\n4\n5\n6\n7\n"; got != want {
			t.Errorf("%s: a refused plan changed the file: %q", tc.name, got)
		}
	}
}
