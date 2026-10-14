package read

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
)

func longLineFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	long := strings.Repeat("x", 300) + "NEEDLE" + strings.Repeat("y", 94)
	body := "short one\n" + long + "\nshort three\n"
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// ADR-137. A read of a 400-character line paid for all of it to show the 40
// characters round a match. With a width the line is cut to a window round the
// match; what was cut was not shown, so the line is not recorded as read, and
// its neighbours, shown whole, are.
func TestAnOverlongLineIsCutToAWindowAndLicensesNothing(t *testing.T) {
	root := longLineFixture(t)
	sp, err := ParseSpec("a.txt:/NEEDLE/")
	if err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	observed, problems := Run(&sb, root, []Spec{sp}, Options{Numbers: true, Context: 1, MaxCols: 40})
	out := sb.String()
	if problems != 0 {
		t.Fatalf("a cut the caller asked for was reported as a problem (%d):\n%s", problems, out)
	}
	if !strings.Contains(out, "NEEDLE") || !strings.Contains(out, "[cols ") || !strings.Contains(out, "of 400]") {
		t.Errorf("the cut line does not show the match and its columns:\n%s", out)
	}
	for _, l := range strings.Split(out, "\n") {
		if len(l) > 120 {
			t.Errorf("a served line is %d bytes, want it cut to a window:\n%s", len(l), l)
		}
	}
	if !strings.Contains(out, "short one") || !strings.Contains(out, "short three") {
		t.Errorf("the neighbours within the width were not served whole:\n%s", out)
	}
	got := observed["a.txt"].Spans
	want := [][2]int{{1, 1}, {3, 3}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("recorded spans = %v, want %v (the cut line 2 left out)", got, want)
	}
	// Without the flag the same read serves line 2 whole and records it.
	var whole strings.Builder
	obs2, _ := Run(&whole, root, []Spec{sp}, Options{Numbers: true, Context: 1})
	if strings.Contains(whole.String(), "[cols ") {
		t.Errorf("a read without --max-cols cut a line:\n%s", whole.String())
	}
	if !reflect.DeepEqual(obs2["a.txt"].Spans, [][2]int{{1, 3}}) {
		t.Errorf("a whole read recorded %v, want [[1 3]]", obs2["a.txt"].Spans)
	}
}

// The window is N characters, centred on the first match and clamped to the
// line, cut on runes.
func TestTheWindowCentresOnTheMatch(t *testing.T) {
	re := []*regexp.Regexp{regexp.MustCompile("NEEDLE")}
	mid := strings.Repeat("a", 100) + "NEEDLE" + strings.Repeat("b", 100)
	got := window(mid, 20, re)
	if !strings.HasPrefix(got, "…") || !strings.Contains(got, "…  [cols ") || !strings.Contains(got, "NEEDLE") {
		t.Errorf("a mid-line match: %q", got)
	}
	front := window("NEEDLE"+strings.Repeat("b", 100), 20, re)
	if strings.HasPrefix(front, "…") || !strings.HasPrefix(front, "NEEDLE") || !strings.Contains(front, "[cols 1-20 of 106]") {
		t.Errorf("a match at the start must not lead with an ellipsis and must start at column 1: %q", front)
	}
	back := window(strings.Repeat("a", 100)+"NEEDLE", 20, re)
	if !strings.HasPrefix(back, "…") || strings.Contains(strings.TrimSuffix(back[:strings.Index(back, "  [cols")], "NEEDLE"), "…b") || !strings.Contains(back, "of 106]") || !strings.HasSuffix(back[:strings.Index(back, "  [cols")], "NEEDLE") {
		t.Errorf("a match at the end must end the window: %q", back)
	}
	none := window(strings.Repeat("a", 100), 20, nil)
	if strings.HasPrefix(none, "…") || !strings.Contains(none, "[cols 1-20 of 100]") {
		t.Errorf("no pattern must start at column 1: %q", none)
	}
	multi := window(strings.Repeat("é", 100), 10, nil)
	text := multi[:strings.Index(multi, "…  [cols")]
	if !utf8.ValidString(multi) || utf8.RuneCountInString(text) != 10 {
		t.Errorf("a window of 10 characters cut inside a character or miscounted: %q", multi)
	}
}
