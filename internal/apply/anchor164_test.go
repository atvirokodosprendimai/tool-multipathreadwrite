package apply

import (
	"strings"
	"testing"
)

// ADR-164 T1. A multi-line replace refused for having no anchor= says, when it
// carries sha=, that sha= does not stand in: a field report's caller gave a
// matching sha= and took it for the guard. Without sha= the refusal is as
// before, so the clause is evidence of the case, not noise on every refusal.
func TestAShaGuardDoesNotStandInForAnAnchor(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", "1\n2\n3\n4\n5\n6\n7\n")
	sha := shaOfFile(t, root, "f.txt")

	for _, tc := range []struct {
		name   string
		sha    string
		clause bool
	}{
		{"with sha=", sha, true},
		{"without sha=", "", false},
	} {
		res, err := Apply(root, []Input{
			{Path: "f.txt", Start: 3, End: 5, Op: "replace", Body: []string{"NEW"}, SHA: tc.sha, Lines: -1, Index: 0},
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
		if has := strings.Contains(got, "sha= checks that the whole file is unchanged, not which lines this range covers"); has != tc.clause {
			t.Errorf("%s: clause present %v, want %v: %s", tc.name, has, tc.clause, got)
		}
		if got, want := read(t, root, "f.txt"), "1\n2\n3\n4\n5\n6\n7\n"; got != want {
			t.Errorf("%s: a refused plan changed the file: %q", tc.name, got)
		}
	}
}
