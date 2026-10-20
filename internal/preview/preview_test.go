package preview

import (
	"strings"
	"testing"
)

// ADR-164 Decision 2. One line replaced by one line is a single `~` row with
// the change marked in place; the marked span runs from the first to the last
// byte that differs, so a second match on the same line is inside it, and a
// span over 160 characters keeps both its ends.
func TestALineChangeIsMarkedInPlace(t *testing.T) {
	cases := []struct {
		name          string
		before, after string
		want          string
	}{
		{"one word", ".a { color: red; }", ".a { color: blue; }", "   ~ 7| .a { color: [-red-]{+blue+}; }"},
		{"an insertion marks only the new text", "ab", "aXb", "   ~ 7| a{+X+}b"},
		{"a deletion marks only the old text", "aXb", "ab", "   ~ 7| a[-X-]b"},
		{"two matches on one line stay inside one span", "red and red", "blue and blue", "   ~ 7| [-red and red-]{+blue and blue+}"},
		{"context is 40 characters each side", strings.Repeat("a", 50) + "X" + strings.Repeat("b", 50), strings.Repeat("a", 50) + "Y" + strings.Repeat("b", 50),
			"   ~ 7| …" + strings.Repeat("a", 40) + "[-X-]{+Y+}" + strings.Repeat("b", 40) + "…"},
		{"a multi-byte character is never cut", "naïve ŝ x", "naïve ŝ y", "   ~ 7| naïve ŝ [-x-]{+y+}"},
	}
	for _, c := range cases {
		got := Lines([]string{c.before}, []string{c.after}, 7)
		if len(got) != 1 || got[0] != c.want {
			t.Errorf("%s: got %q, want [%q]", c.name, got, c.want)
		}
	}

	// A span over 160 characters keeps its first and last 70 with the count
	// between: both matches below are in the span, and both are visible.
	old := "first" + strings.Repeat("m", 200) + "last"
	got := Lines([]string{"x" + old + "y"}, []string{"x" + strings.ToUpper(old) + "y"}, 3)
	if len(got) != 1 {
		t.Fatalf("a long span: %q", got)
	}
	for _, s := range []string{"[-" + old[:70] + "…[69 chars]…" + old[len(old)-70:] + "-]", "{+FIRST", "LAST+}", "x[-", "-]{+", "+}y"} {
		if !strings.Contains(got[0], s) {
			t.Errorf("a long span lacks %q:\n%s", s, got[0])
		}
	}

	// A change past column 200 is shown.
	long := strings.Repeat("0123456789", 30)
	got = Lines([]string{long + "red tail"}, []string{long + "blue tail"}, 1)
	if len(got) != 1 || !strings.Contains(got[0], "[-red-]{+blue+}") || len(got[0]) > 200 {
		t.Errorf("a change past column 200: %q", got)
	}
}

// ADR-164 Decision 2. Anything else is the old lines under `-` and the new
// under `+`; a line paired with the line at the same index is marked and cut
// by the same rule, so every change on it shows however far apart, and a line
// with no counterpart is cut to its first 200 characters.
func TestAMultiLineChangeShowsOldThenNew(t *testing.T) {
	// Two adjacent 300-character lines, each changed at column 10 and 290.
	mk := func(a, b string) string {
		return strings.Repeat("x", 10) + a + strings.Repeat("y", 270) + b + strings.Repeat("z", 10)
	}
	before := []string{mk("old", "old"), mk("OLD", "OLD")}
	after := []string{mk("new", "new"), mk("NEW", "NEW")}
	got := Lines(before, after, 10)
	if len(got) != 4 {
		t.Fatalf("got %d rows, want 2 old then 2 new:\n%s", len(got), strings.Join(got, "\n"))
	}
	for i, prefix := range []string{"   - 10| ", "   - 11| ", "   + 10| ", "   + 11| "} {
		if !strings.HasPrefix(got[i], prefix) {
			t.Errorf("row %d = %q, want prefix %q", i, got[i], prefix)
		}
	}
	for i, want := range [][2]string{{"[-old", "old-]"}, {"[-OLD", "OLD-]"}, {"{+new", "new+}"}, {"{+NEW", "NEW+}"}} {
		if !strings.Contains(got[i], want[0]) || !strings.Contains(got[i], want[1]) || !strings.Contains(got[i], "…[") {
			t.Errorf("row %d lacks both changes (%v):\n%s", i, want, got[i])
		}
		if len(got[i]) > 300 {
			t.Errorf("row %d is not cut: %d bytes", i, len(got[i]))
		}
	}

	// One line becoming two: no pair for the second, cut to 200 characters.
	extra := strings.Repeat("e", 250)
	got = Lines([]string{"a"}, []string{"b", extra}, 5)
	want := []string{"   - 5| [-a-]", "   + 5| {+b+}", "   + 6| " + strings.Repeat("e", 200) + "…"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("1 → 2 lines:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	// Two lines becoming one: the second old line stands alone.
	got = Lines([]string{"a", "b"}, []string{"c"}, 1)
	want = []string{"   - 1| [-a-]", "   - 2| b", "   + 1| {+c+}"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("2 → 1 lines:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
