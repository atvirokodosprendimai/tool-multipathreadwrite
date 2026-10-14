package mcp

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// ckOpen is a checkpoint's open marker: its id and the span it brackets.
var ckOpen = regexp.MustCompile(`^-- ck (\S+) open lines (\d+)-(\d+) `)

// checkpointSpans is every span the served text's checkpoints bracket.
func checkpointSpans(text string) [][2]int {
	var out [][2]int
	for _, line := range strings.Split(text, "\n") {
		if m := ckOpen.FindStringSubmatch(line); m != nil {
			a, _ := strconv.Atoi(m[2])
			b, _ := strconv.Atoi(m[3])
			out = append(out, [2]int{a, b})
		}
	}
	return out
}

// bracketed reports whether any checkpoint in text brackets line.
func bracketed(text string, line int) bool {
	for _, sp := range checkpointSpans(text) {
		if sp[0] <= line && line <= sp[1] {
			return true
		}
	}
	return false
}

// ADR-154. mrw_read takes max_cols as `mrw read --max-cols` does, and a line it
// served cut sits in no checkpoint: acknowledging every id the answer carries
// licenses the lines served whole and never the cut one (ADR-137 on the CLI).
func TestACutLineIsOutsideEveryCheckpoint(t *testing.T) {
	long := strings.Repeat("x", 120) + " needle " + strings.Repeat("y", 120)
	root, path := checkout(t, "a.txt", "one\n"+long+"\nthree\nfour\n")
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{path}, "max_cols": 20})
	txt := served0(t, res)
	if res["isError"] == true || !strings.Contains(txt, "[cols ") || strings.Contains(txt, long) {
		t.Fatalf("max_cols 20 did not cut line 2: %q", txt)
	}
	if bracketed(txt, 2) {
		t.Errorf("a checkpoint brackets the cut line 2: %q", txt)
	}
	if !bracketed(txt, 1) || !bracketed(txt, 3) || !bracketed(txt, 4) {
		t.Errorf("the lines served whole are not all bracketed: %q", txt)
	}
	if !strings.Contains(txt, "-- not licensed: line(s) 2 were served cut by max_cols") {
		t.Errorf("the answer does not name the cut line: %q", txt)
	}
	acks := checkpointsIn(txt)
	if len(acks) == 0 {
		t.Fatalf("no checkpoint at all: %q", txt)
	}
	if r := call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 2 replace\nTWO\n", "ack": acks}); r["isError"] != true || !strings.Contains(firstText(t, r), "has not been read") {
		t.Errorf("a write to the cut line was not refused as unread after every ack: %v", r)
	}
	if r := call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 3 replace\nTHREE\n"}); r["isError"] == true {
		t.Errorf("a write to a line served whole was refused: %s", firstText(t, r))
	}
}

// ADR-154 Decision 3: the cut is known from what the read recorded, never from
// the marker text, which a file can hold itself.
func TestALineThatSaysColsIsNotTakenForACutOne(t *testing.T) {
	root, path := checkout(t, "a.txt", "one\nshort  [cols 1-5 of 9]\nthree\n")
	txt := served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{path}, "max_cols": 80}))
	if !bracketed(txt, 2) || strings.Contains(txt, "-- not licensed:") {
		t.Fatalf("a whole line holding the marker text was left out: %q", txt)
	}
	acks := checkpointsIn(txt)
	if r := call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 2 replace\nTWO\n", "ack": acks}); r["isError"] == true {
		t.Errorf("its write was refused: %s", firstText(t, r))
	}
}

// ADR-154 Decision 1: a width cannot be negative, as on the CLI.
func TestANegativeMaxColsIsRefused(t *testing.T) {
	root, path := checkout(t, "a.txt", "one\n")
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{path}, "max_cols": -1})
	if res["isError"] != true || !strings.Contains(firstText(t, res), "max_cols -1") || !strings.Contains(firstText(t, res), "negative") {
		t.Errorf("max_cols -1: %v", res)
	}
	// Beside count, which serves no line, a width has nothing to cut, as
	// max_lines has nothing to cap there (ADR-153).
	res = call(t, root, "mrw_read", map[string]any{"grep": "one", "count": true, "max_cols": 5})
	if res["isError"] != true || !strings.Contains(firstText(t, res), "max_cols has nothing to cut") {
		t.Errorf("max_cols beside count: %v", res)
	}
}

// ADR-154 Decision 1 and 2 on the paged path: a read too large for one answer
// is paged by firstPage, which reads again; that read is cut too, and its cut
// lines are in no checkpoint.
func TestAPageCutByMaxColsLeavesTheCutLinesOut(t *testing.T) {
	restore := MaxResultChars
	t.Cleanup(func() { MaxResultChars = restore })
	var b strings.Builder
	long := strings.Repeat("z", 400)
	for i := 1; i <= 600; i++ {
		if i%10 == 0 {
			b.WriteString(long + "\n")
		} else {
			b.WriteString("line " + strconv.Itoa(i) + "\n")
		}
	}
	root, path := checkout(t, "big.txt", b.String())
	MaxResultChars = 6000
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{path}, "max_cols": 20})
	txt := served0(t, res)
	if !strings.Contains(txt, "-- PARTIAL:") {
		t.Fatalf("the read was not paged: %q", txt)
	}
	if !strings.Contains(txt, "[cols ") || strings.Contains(txt, long) {
		t.Fatalf("the page was not cut: %q", txt)
	}
	if bracketed(txt, 10) || !bracketed(txt, 9) || !bracketed(txt, 11) {
		t.Errorf("the page's cut line 10 is bracketed, or its neighbours are not: %q", txt)
	}
}
