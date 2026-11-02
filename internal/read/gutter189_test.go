package read

import (
	"fmt"
	"strings"
	"testing"
)

// ADR-189: a served line is one space, its number right-aligned to the digits
// of the largest line its file block's spans plan to serve (ADR-161's line after
// a range included), then "| ". The width is fixed before --max-lines cuts, and
// each file block computes its own.
func TestAReadNumbersLinesAsWideAsTheLargestItPlans(t *testing.T) {
	root := t.TempDir()
	numbered := func(n int) string {
		var b strings.Builder
		for i := 1; i <= n; i++ {
			fmt.Fprintf(&b, "l%d\n", i)
		}
		return b.String()
	}
	writeFile(t, root, "nine.txt", numbered(9))
	writeFile(t, root, "twelve.txt", numbered(12))
	writeFile(t, root, "long.txt", numbered(120))
	two := 2
	for _, c := range []struct {
		name  string
		opt   Options
		specs []string
		want  []string // lines that must be served exactly
		not   []string // substrings that must not appear
	}{
		{"a 9-line file read whole is one digit wide", Options{Numbers: true}, []string{"nine.txt"},
			[]string{" 1| l1", " 9| l9"}, []string{"    1| "}},
		{"a low range of a long file is as wide as its planned end, not the file", Options{Numbers: true, NextLine: true}, []string{"long.txt:5-6"},
			[]string{" 5| l5", " 7| l7"}, []string{"  5| "}},
		{"the line after a range can widen the block", Options{Numbers: true, NextLine: true}, []string{"twelve.txt:1-9"},
			[]string{"  1| l1", " 10| l10"}, nil},
		{"a span --max-lines withholds still sets the width", Options{Numbers: true, NextLine: true, MaxLines: &two}, []string{"long.txt:5-6,118"},
			[]string{"   5| l5", "   6| l6"}, []string{"118| "}},
		// The wide block first: a width carried over from one block to the next only grows, so
		// the narrow block after it is what tells a per-block width from a shared one.
		{"each file block has its own width", Options{Numbers: true}, []string{"long.txt:100", "nine.txt:2"},
			[]string{" 100| l100", " 2| l2"}, []string{"  2| l2"}},
		{"-N serves no gutter", Options{}, []string{"nine.txt:3"},
			[]string{"l3"}, []string{"| "}},
	} {
		out, _ := run(t, root, c.opt, c.specs...)
		lines := strings.Split(out, "\n")
		for _, w := range c.want {
			found := false
			for _, l := range lines {
				if l == w {
					found = true
				}
			}
			if !found {
				t.Errorf("%s: no line %q in\n%s", c.name, w, out)
			}
		}
		for _, n := range c.not {
			if strings.Contains(out, n) {
				t.Errorf("%s: %q in\n%s", c.name, n, out)
			}
		}
	}
}
