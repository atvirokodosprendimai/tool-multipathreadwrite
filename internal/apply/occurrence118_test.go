package apply

import (
	"regexp"
	"strings"
	"testing"
)

// ADR-118. A start pattern that matches several lines may pick one with
// occurrence=N, provided every match before the Nth has been served: the ledger
// is per sha, so the caller has counted the matches in the file as it is.
const threeFuncs = "package x\n\nfunc X() int {\n\treturn 1\n}\n\nfunc X() int {\n\treturn 2\n}\n\nfunc X() int {\n\treturn 3\n}\n"

func occurrenceInput(n int, body ...string) Input {
	return Input{
		Path: "x.go", Op: "replace", Lines: -1, Body: body, Occurrence: n, Anchor: "func X",
		StartPat: regexp.MustCompile(`^func X`), EndPat: regexp.MustCompile(`^}`),
	}
}

func TestAnOccurrencePicksTheNthStartMatch(t *testing.T) {
	root := t.TempDir()
	write(t, root, "x.go", threeFuncs)
	ledger := map[string]Seen{"x.go": {SHA: shaOfFile(t, root, "x.go")}} // the whole file served
	res, err := Apply(root, []Input{occurrenceInput(2, "func X() int {", "\treturn 20", "}")}, Options{Seen: ledger})
	if err != nil || res.Failed != 0 {
		t.Fatalf("Apply: %v failed=%d %+v", err, res.Failed, res.Hunks)
	}
	want := strings.Replace(threeFuncs, "\treturn 2\n", "\treturn 20\n", 1)
	if got := read(t, root, "x.go"); got != want {
		t.Errorf("occurrence=2 changed the wrong match:\n%s", got)
	}
}

func TestAnOccurrencePastTheMatchCountIsRefused(t *testing.T) {
	root := t.TempDir()
	write(t, root, "x.go", threeFuncs)
	res, err := Apply(root, []Input{occurrenceInput(4, "x")}, Options{Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 || !strings.Contains(res.Hunks[0].Reason, "3, 7, 11") {
		t.Errorf("occurrence=4 of three: failed=%d reason %q, want a refusal naming the matches", res.Failed, res.Hunks[0].Reason)
	}
	if read(t, root, "x.go") != threeFuncs {
		t.Error("a refused occurrence changed the file")
	}
}

func TestAnOccurrenceWhoseEarlierMatchesWereNotServedIsRefused(t *testing.T) {
	root := t.TempDir()
	write(t, root, "x.go", threeFuncs)
	sha := shaOfFile(t, root, "x.go")
	// Lines 7-9 — the second function — served; the first match, line 3, not.
	res, err := Apply(root, []Input{occurrenceInput(2, "func X() int {", "\treturn 20", "}")},
		Options{Seen: map[string]Seen{"x.go": {SHA: sha, Spans: [][2]int{{7, 10}}}}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 || !strings.Contains(res.Hunks[0].Reason, "3") || !strings.Contains(res.Hunks[0].Reason, "occurrence") {
		t.Errorf("an unread earlier match: failed=%d reason %q, want a refusal naming line 3", res.Failed, res.Hunks[0].Reason)
	}
	if read(t, root, "x.go") != threeFuncs {
		t.Error("a refused occurrence changed the file")
	}
	// The pair: once line 3 is served too, it applies.
	res, err = Apply(root, []Input{occurrenceInput(2, "func X() int {", "\treturn 20", "}")},
		Options{Seen: map[string]Seen{"x.go": {SHA: sha, Spans: [][2]int{{3, 3}, {7, 10}}}}})
	if err != nil || res.Failed != 0 {
		t.Errorf("with every earlier match served: %v failed=%d %+v", err, res.Failed, res.Hunks)
	}
}

func TestTheSeveralMatchesRefusalNamesOccurrence(t *testing.T) {
	root := t.TempDir()
	write(t, root, "x.go", threeFuncs)
	res, err := Apply(root, []Input{occurrenceInput(0, "x")}, Options{Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 || !strings.Contains(res.Hunks[0].Reason, "occurrence=N") {
		t.Errorf("the several-matches refusal: %q, want it to name occurrence=N", res.Hunks[0].Reason)
	}
}

// TestAnOccurrenceAfterAWriteNeedsTheMatchesRead: a file mrw just wrote is
// wholly licensed (ADR-005) but the caller was shown none of it, so the
// earlier matches still need a read (ADR-118; Zy, "strict: always served").
func TestAnOccurrenceAfterAWriteNeedsTheMatchesRead(t *testing.T) {
	root := t.TempDir()
	write(t, root, "x.go", threeFuncs)
	sha := shaOfFile(t, root, "x.go")
	in := occurrenceInput(2, "func X() int {", "\treturn 20", "}")
	res, err := Apply(root, []Input{in}, Options{DryRun: true, Seen: map[string]Seen{"x.go": {SHA: sha, Written: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 || !strings.Contains(res.Hunks[0].Reason, "lines 3") {
		t.Fatalf("after a write, no match read: failed=%d reason %q, want a refusal naming line 3", res.Failed, res.Hunks[0].Reason)
	}
	res, err = Apply(root, []Input{in}, Options{DryRun: true, Seen: map[string]Seen{"x.go": {SHA: sha, Written: true, Shown: [][2]int{{3, 3}}}}})
	if err != nil || res.Failed != 0 {
		t.Errorf("after a write and a read of the earlier match: %v failed=%d %+v", err, res.Failed, res.Hunks)
	}
}

// --force skips the earlier-matches rule as it skips the ledger; two hunks
// pick two matches of one pattern in one plan, each against the original file;
// and the receipt names which match each hunk picked.
func TestAnOccurrenceUnderForceAndTwiceInOnePlan(t *testing.T) {
	root := t.TempDir()
	write(t, root, "x.go", threeFuncs)
	ret := regexp.MustCompile(`^\treturn`)
	res, err := Apply(root, []Input{
		{Path: "x.go", Op: "replace", Lines: -1, Body: []string{"\treturn 10"}, StartPat: ret, Occurrence: 1},
		{Path: "x.go", Op: "replace", Lines: -1, Body: []string{"\treturn 30"}, StartPat: ret, Occurrence: 3, Index: 1},
	}, Options{Force: true, Seen: map[string]Seen{"x.go": {SHA: shaOfFile(t, root, "x.go"), Spans: [][2]int{}}}})
	if err != nil || res.Failed != 0 {
		t.Fatalf("Apply: %v failed=%d %+v", err, res.Failed, res.Hunks)
	}
	got := read(t, root, "x.go")
	if !strings.Contains(got, "return 10") || !strings.Contains(got, "return 2\n") || !strings.Contains(got, "return 30") {
		t.Errorf("the two picks landed wrong:\n%s", got)
	}
	if res.Hunks[0].Addr != `/^\treturn/ occurrence=1` || res.Hunks[1].Addr != `/^\treturn/ occurrence=3` {
		t.Errorf("the receipt does not say which match each picked: %q %q", res.Hunks[0].Addr, res.Hunks[1].Addr)
	}
}
