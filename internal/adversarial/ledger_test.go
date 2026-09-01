package adversarial

import (
	"io"
	"strings"
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

// ADR-002 says mrw will not edit a file it has not seen, and its REASON is that
// "a range address like 42-58 only means something in the version of the file
// those numbers were counted in". --stat renders no content at all — it is
// documented as "ask for the fact, not the artifact" — so after one, mrw holds
// a hash and the caller has counted nothing. It must license no edit.
func TestAStatOnlyReadLicensesNothing(t *testing.T) {
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

	if res.Failed != 1 {
		t.Fatalf("a --stat read authorised an edit to line 20: failed=%d, applied=%v", res.Failed, res.Applied)
	}
	if got := readFile(t, root, "big.go"); got != long() {
		t.Error("the file was written despite the refusal")
	}
}

// A ranged read licenses the lines it SERVED, and no others. Reading line 1
// tells the caller nothing about line 40, so an address there is written
// against a picture they do not have.
func TestARangedReadLicensesOnlyTheLinesItServed(t *testing.T) {
	root := tree(t, map[string]string{"big.go": long()})

	observed, _ := read.Run(io.Discard, root,
		[]read.Spec{{Path: "big.go", Ranges: []read.Range{{Start: 1, End: 5}}}}, read.Options{})

	inside, err := apply.Apply(root, []apply.Input{{
		Path: "big.go", Start: 3, End: 3, Op: "replace",
		Body: []string{"rewritten"}, Lines: unset,
	}}, apply.Options{Seen: observed})
	if err != nil {
		t.Fatal(err)
	}
	if inside.Failed != 0 {
		t.Errorf("an edit inside the served range was refused: %s", inside.Hunks[0].Reason)
	}

	root2 := tree(t, map[string]string{"big.go": long()})
	observed2, _ := read.Run(io.Discard, root2,
		[]read.Spec{{Path: "big.go", Ranges: []read.Range{{Start: 1, End: 5}}}}, read.Options{})

	outside, err := apply.Apply(root2, []apply.Input{{
		Path: "big.go", Start: 40, End: 40, Op: "replace",
		Body: []string{"rewritten"}, Lines: unset,
	}}, apply.Options{Seen: observed2})
	if err != nil {
		t.Fatal(err)
	}
	if outside.Failed != 1 {
		t.Fatalf("reading lines 1-5 authorised an edit to line 40: failed=%d", outside.Failed)
	}
	if !strings.Contains(outside.Hunks[0].Reason, "has not been read") {
		t.Errorf("the refusal does not say the lines were never read: %s", outside.Hunks[0].Reason)
	}
	if got := readFile(t, root2, "big.go"); got != long() {
		t.Error("the file was written despite the refusal")
	}
}

// A whole-file read licenses the whole file, which is the ordinary case and
// must not become expensive to express.
func TestAWholeFileReadLicensesTheWholeFile(t *testing.T) {
	root := tree(t, map[string]string{"big.go": long()})

	observed, _ := read.Run(io.Discard, root, []read.Spec{{Path: "big.go"}}, read.Options{})

	res, err := apply.Apply(root, []apply.Input{{
		Path: "big.go", Start: 40, End: 40, Op: "replace",
		Body: []string{"rewritten"}, Lines: unset,
	}}, apply.Options{Seen: observed})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Fatalf("a whole-file read did not license line 40: %s", res.Hunks[0].Reason)
	}
	if got := readFile(t, root, "big.go"); !strings.Contains(got, "rewritten") {
		t.Error("the run reported ok and wrote nothing")
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
	if got := readFile(t, root, "a.go"); !strings.HasPrefix(got, "package q") {
		t.Errorf("the write reported ok but the file is unchanged:\n%s", got)
	}
}

// And the guard itself still has to bite: a file mrw has never read is refused,
// whatever spelling it arrives under.
func TestAnUnreadFileIsStillRefused(t *testing.T) {
	root := tree(t, map[string]string{"a.go": goFile})

	res, err := apply.Apply(root, []apply.Input{{
		Path: "a.go", Start: 1, End: 1, Op: "replace",
		Body: []string{"package q"}, Lines: unset,
	}}, apply.Options{Seen: map[string]apply.Seen{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Errorf("an unread file was edited: failed=%d, applied=%v", res.Failed, res.Applied)
	}
	if got := readFile(t, root, "a.go"); got != goFile {
		t.Errorf("the refusal still wrote the file:\n%s", got)
	}
}

// The reviewer's finding, as a test: --max-lines is the flag a caller reaches
// for on a big file, which is exactly when they cannot count the lines they
// were not shown. A truncated read must observe what it PRINTED, not what it
// was asked for — the request shape says nothing about what the caller saw.
func TestATruncatedReadLicensesOnlyTheLinesItPrinted(t *testing.T) {
	root := tree(t, map[string]string{"big.go": long()})

	observed, _ := read.Run(io.Discard, root,
		[]read.Spec{{Path: "big.go"}}, read.Options{MaxLines: 5})

	res, err := apply.Apply(root, []apply.Input{{
		Path: "big.go", Start: 40, End: 40, Op: "replace",
		Body: []string{"rewritten"}, Lines: unset,
	}}, apply.Options{Seen: observed})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Fatalf("a read that withheld 35 lines authorised an edit to line 40: failed=%d, applied=%v",
			res.Failed, res.Applied)
	}
	if got := readFile(t, root, "big.go"); got != long() {
		t.Error("the file was written despite the refusal")
	}
}

// And the other direction: reading a file WHOLE and then reading part of it
// again must not downgrade the observation. Reading more thoroughly cannot
// observe less.
func TestAWholeReadIsNotDowngradedByALaterRangedRead(t *testing.T) {
	root := tree(t, map[string]string{"big.go": long()})

	observed, _ := read.Run(io.Discard, root,
		[]read.Spec{{Path: "big.go"}, {Path: "big.go", Ranges: []read.Range{{Start: 1, End: 2}}}},
		read.Options{})

	res, err := apply.Apply(root, []apply.Input{{
		Path: "big.go", Start: 40, End: 40, Op: "replace",
		Body: []string{"rewritten"}, Lines: unset,
	}}, apply.Options{Seen: observed})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Fatalf("reading the whole file and then part of it refused line 40: %s", res.Hunks[0].Reason)
	}
}
