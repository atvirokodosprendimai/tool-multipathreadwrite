package adversarial

import (
	"io"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
)

// long is a file whose middle a caller could not have counted from a one-line
// read.
func long() string {
	s := ""
	for i := 1; i <= 40; i++ {
		s += "line\n"
	}
	return s
}

// SCOPE FINDING, for M rather than a defect to fix here.
//
// ADR-002 says "mrw will not edit a file it has not seen", and its reason is
// that "a range address like 42-58 only means something in the version of the
// file those numbers were counted in". The ledger implements the first
// sentence: read.Run records the whole-file sha for every spec it serves,
// BEFORE deciding how much of the file to render. So --stat — documented as
// "ask for the fact, not the artifact", and printing no content at all — marks
// the file seen, and an edit to any line is then authorised.
//
// mrw has indeed seen the file. The CALLER has not, and the caller is who
// counts lines. Whether that gap matters is a decision about what the ledger is
// for, so this test pins today's answer rather than asserting a new one: it
// fails the day the behaviour changes, in either direction.
func TestKnownGap_AStatOnlyReadLicensesAnEdit(t *testing.T) {
	root := tree(t, map[string]string{"big.go": long()})

	observed, problems := read.Run(io.Discard, root, []read.Spec{{Path: "big.go"}}, read.Options{Stat: true})
	if problems != 0 {
		t.Fatalf("stat read reported %d problem(s)", problems)
	}

	res, err := apply.Apply(root, []apply.Input{{
		Path: "big.go", Start: 20, End: 20, Op: "replace",
		Body: []string{"rewritten"}, Lines: unset,
	}}, apply.Options{Seen: observed})
	if err != nil {
		t.Fatal(err)
	}

	if res.Failed != 0 {
		t.Errorf("a --stat read no longer licenses an edit — that is arguably the better rule, "+
			"but ADR-002 and the CLI's own help text still describe the old one: %s", res.Hunks[0].Reason)
	}
}

// The same gap, narrower: one line served, an edit thirty-nine lines away.
func TestKnownGap_AOneLineReadLicensesAnEditElsewhereInTheFile(t *testing.T) {
	root := tree(t, map[string]string{"big.go": long()})

	observed, _ := read.Run(io.Discard, root,
		[]read.Spec{{Path: "big.go", Ranges: []read.Range{{Start: 1, End: 1}}}}, read.Options{})

	res, err := apply.Apply(root, []apply.Input{{
		Path: "big.go", Start: 40, End: 40, Op: "replace",
		Body: []string{"rewritten"}, Lines: unset,
	}}, apply.Options{Seen: observed})
	if err != nil {
		t.Fatal(err)
	}

	if res.Failed != 0 {
		t.Errorf("reading line 1 no longer licenses an edit to line 40: %s", res.Hunks[0].Reason)
	}
}

// The ledger is keyed by the path the caller typed, and two spellings of one
// file are one file. Refusing here was a false alarm, and a guard that cries
// wolf is a guard people pass --force to.
func TestTwoSpellingsOfOnePathAreOneFile(t *testing.T) {
	root := tree(t, map[string]string{"a.go": goFile})

	observed, _ := read.Run(io.Discard, root, []read.Spec{{Path: "a.go"}}, read.Options{})

	res, err := apply.Apply(root, []apply.Input{{
		Path: "./a.go", Start: 1, End: 1, Op: "replace",
		Body: []string{"package q"}, Lines: unset,
	}}, apply.Options{Seen: observed})
	if err != nil {
		t.Fatal(err)
	}

	if res.Failed != 0 {
		t.Errorf("read as %q then written as %q was refused as unseen: %s", "a.go", "./a.go", res.Hunks[0].Reason)
	}
}

// And the guard itself still has to bite: a file mrw has never read is refused,
// whatever spelling it arrives under.
func TestAnUnreadFileIsStillRefused(t *testing.T) {
	root := tree(t, map[string]string{"a.go": goFile})

	res, err := apply.Apply(root, []apply.Input{{
		Path: "a.go", Start: 1, End: 1, Op: "replace",
		Body: []string{"package q"}, Lines: unset,
	}}, apply.Options{Seen: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Errorf("an unread file was edited: failed=%d, applied=%v", res.Failed, res.Applied)
	}
}
