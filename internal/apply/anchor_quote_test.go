package apply

import (
	"strings"
	"testing"
)

// ADR-028: an anchor= written `anchor="    "timeout": 30,"` reaches apply as
// `    timeout: 30,`, because the header splitter takes every " as quoting. When
// the line holds the anchor once its quotes are gone, and no line holds it as
// written, the refusal says so and names the two spellings that keep a quote:
// single quotes, or \" inside double. An anchor that misses for any other
// reason — one that matches nothing, or one another line holds, a wrong range —
// gets no such hint (the go-reviewer pass on this change).
func TestAnAnchorThatLostItsQuotesSaysHowToKeepThem(t *testing.T) {
	const quoted = "{\n    \"timeout\": 30,\n    \"retries\": 2\n}\n"
	for _, op := range []string{"replace", "insert-after"} {
		for _, c := range []struct {
			file, anchor string
			hint         bool
		}{
			{quoted, "    timeout: 30,", true},
			{quoted, "nope", false},
			{quoted + "    timeout: 30,\n", "    timeout: 30,", false},
		} {
			root := t.TempDir()
			write(t, root, "c.json", c.file)
			res, err := Apply(root, []Input{
				{Path: "c.json", Start: 2, End: 2, Op: op, Body: []string{"x"}, Anchor: c.anchor, Lines: -1, Index: 0},
			}, Options{})
			if err != nil {
				t.Fatal(err)
			}
			r := res.Hunks[0].Reason
			if res.Failed != 1 || !strings.Contains(r, "not in line 2") {
				t.Fatalf("%s %q: failed=%d reason %q, want the anchor refusal", op, c.anchor, res.Failed, r)
			}
			got := strings.Contains(r, `anchor='…'`) && strings.Contains(r, `\"`)
			if got != c.hint {
				t.Errorf("%s %q: reason %q, hint present %v, want %v", op, c.anchor, r, got, c.hint)
			}
		}
	}
}
