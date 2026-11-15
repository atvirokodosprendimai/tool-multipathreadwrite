package apply

import (
	"strings"
	"testing"
)

// ADR-197: an anchor= that is not in the addressed line is usually a wrong
// range, and the refusal quoted the wrong line without saying where the right
// one is. It now ends with the lines that hold the anchor as written (a
// substring, the test the anchor itself gets), the first five and a count of
// the rest, or says no line does. The #589 case keeps its quote advice alone:
// there the addressed line is the right one.
func TestAnAnchorMissNamesTheLinesThatHoldIt(t *testing.T) {
	lines := func(n int, hold ...int) string {
		at := map[int]bool{}
		for _, h := range hold {
			at[h] = true
		}
		var b strings.Builder
		for i := 1; i <= n; i++ {
			if at[i] {
				b.WriteString("  retarget here\n")
			} else {
				b.WriteString("filler\n")
			}
		}
		return b.String()
	}
	for _, c := range []struct {
		name, file, anchor, end string
		absent                  []string
	}{
		{"one other line", lines(5, 4), "target", "; the anchor is in line 4", nil},
		{"three other lines", lines(9, 4, 7, 9), "target", "; the anchor is in lines 4, 7, 9", nil},
		{"seven other lines", lines(9, 3, 4, 5, 6, 7, 8, 9), "target", "; the anchor is in lines 3, 4, 5, 6, 7 and 2 more", nil},
		{"no line", lines(5), "target", "; no line of f.txt holds it", nil},
		{"the anchor lost its quotes", "{\n    \"timeout\": 30,\n}\n", "    timeout: 30,", `inside double quotes`,
			[]string{"no line of", "the anchor is in"}},
	} {
		root := t.TempDir()
		write(t, root, "f.txt", c.file)
		res, err := Apply(root, []Input{
			{Path: "f.txt", Start: 2, End: 2, Op: "replace", Body: []string{"x"}, Anchor: c.anchor, Lines: -1, Index: 0},
		}, Options{})
		if err != nil {
			t.Fatal(err)
		}
		r := res.Hunks[0].Reason
		if res.Failed != 1 || !strings.Contains(r, "not in line 2: ") {
			t.Fatalf("%s: failed=%d reason %q, want the anchor refusal with its quoted line", c.name, res.Failed, r)
		}
		if !strings.HasSuffix(r, c.end) {
			t.Errorf("%s: reason %q, want it to end %q", c.name, r, c.end)
		}
		for _, a := range c.absent {
			if strings.Contains(r, a) {
				t.Errorf("%s: reason %q holds %q, which belongs to another case", c.name, r, a)
			}
		}
	}

	// An anchor that matches is not a refusal, wherever else it also is.
	root := t.TempDir()
	write(t, root, "f.txt", lines(5, 2, 4))
	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 2, End: 2, Op: "replace", Body: []string{"x"}, Anchor: "target", Lines: -1, Index: 0},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Errorf("an anchor in its line: failed=%d reason %q, want it applied", res.Failed, res.Hunks[0].Reason)
	}
}
