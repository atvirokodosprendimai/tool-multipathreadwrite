package adversarial

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
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

// A ranged read that matches NOTHING serves nothing, so it must observe
// nothing. Found 2026-09-01 while implementing ADR-007 T2, live on main:
// `mrw read big.txt:/nosuchpattern/` printed "no match", recorded the WHOLE
// FILE, and then licensed an edit to line 40 that nobody had been shown.
//
// The cause is worth keeping because it is invisible in the code: `served`
// was a nil slice when nothing printed, and a nil Spans means "whole file" to
// seen.Observation. Same class as --stat and --max-lines, which ADR-005 closed
// — a third path where the caller saw nothing and the ledger said everything.
func TestAPatternThatMatchesNothingObservesNothing(t *testing.T) {
	root := tree(t, map[string]string{"big.go": long()})

	spec, err := read.ParseSpec("big.go:/nosuchpattern/")
	if err != nil {
		t.Fatal(err)
	}
	observed, problems := read.Run(io.Discard, root, []read.Spec{spec}, read.Options{})
	if problems == 0 {
		t.Error("a pattern matching nothing was not reported")
	}
	if obs, ok := observed["big.go"]; ok && obs.Whole() {
		t.Error("nothing was printed and the whole file was observed")
	}

	res, err := apply.Apply(root, []apply.Input{{
		Path: "big.go", Start: 40, End: 40, Op: "replace",
		Body: []string{"rewritten"}, Lines: unset,
	}}, apply.Options{Seen: observed})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Fatalf("an edit to line 40 was licensed by a read that printed nothing: failed=%d", res.Failed)
	}
}

// ADR-002 and ADR-005 say mrw does not tell you what it has not shown you, and
// `anchor=` was the one guard that did: it was checked ABOVE the ledger, so a
// FAILED anchor quoted a line the caller had never been served.
//
// ⚠ THE FIXTURE SERVES LINE 1 AND ANCHORS LINE 2, and that is not incidental.
// docs/adr/BACKLOG.md:226 pre-registers the trap: a fixture that serves NOTHING
// is refused by the whole-file gate before either guard runs, so it passes with
// the ordering reversed and proves nothing. The file must be partly served.
func TestAFailedAnchorDoesNotReadBackAnUnservedLine(t *testing.T) {
	const secret = "UNSERVED-SENTINEL-42"
	root := tree(t, map[string]string{"f.txt": "public line\n" + secret + "\nthird\n"})

	// Line 1 only.
	observed, _ := read.Run(io.Discard, root,
		[]read.Spec{{Path: "f.txt", Ranges: []read.Range{{Start: 1, End: 1}}}}, read.Options{})

	// EVERY op that carries an anchor, because replace/delete and the two
	// insertions reach the guard by different paths — the first cut of ADR-028
	// fixed one pair and left the other, and claimed "every guard" anyway.
	for _, op := range []string{"replace", "delete", "insert-after", "insert-before"} {
		in := apply.Input{
			Path: "f.txt", Start: 2, End: 2, Op: op,
			Lines: unset, Anchor: "no-such-text",
		}
		if op != "delete" {
			in.Body = []string{"rewritten"}
		}
		res, err := apply.Apply(root, []apply.Input{in}, apply.Options{Seen: observed})
		if err != nil {
			t.Fatalf("%s: %v", op, err)
		}
		if res.Failed != 1 {
			t.Fatalf("%s: failed=%d, want 1 — the hunk addresses a line that was never served", op, res.Failed)
		}
		reason := res.Hunks[0].Reason
		if strings.Contains(reason, secret) {
			t.Errorf("%s: the refusal reads back a line the caller was never served: %s", op, reason)
		}
		if !strings.Contains(reason, "has not been read") {
			t.Errorf("%s: the refusal is not the ledger's: %s", op, reason)
		}
	}

	// The other half, or the fix could be "never check anchors": a line the
	// caller WAS served is still anchor-checked, and its text is still quoted,
	// because they are entitled to it.
	res2, err := apply.Apply(root, []apply.Input{{
		Path: "f.txt", Start: 1, End: 1, Op: "replace",
		Body: []string{"rewritten"}, Lines: unset, Anchor: "no-such-text",
	}}, apply.Options{Seen: observed})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Failed != 1 {
		t.Fatalf("failed=%d, want 1 — the anchor does not match line 1", res2.Failed)
	}
	if !strings.Contains(res2.Hunks[0].Reason, "public line") {
		t.Errorf("a served line's anchor failure no longer quotes it: %s", res2.Hunks[0].Reason)
	}
}

// TestAnAliasSpellingIsTheSameFileToThePerLineLedger pins ADR-029: a file is one
// observation to the per-line gate whatever the plan calls it. Both halves live
// here on purpose — a test that only asserts the alias write is REFUSED is green
// against a fix that refuses every alias, which is issue #47 undone.
func TestAnAliasSpellingIsTheSameFileToThePerLineLedger(t *testing.T) {
	const secret = "UNSERVED-SENTINEL-29"

	// The alias is created by the test rather than asserted about the platform:
	// Windows CI may refuse to make a symlink, and a failure there would be
	// about the harness rather than about the ledger.
	aliased := func(t *testing.T) (root, real, alias string) {
		t.Helper()
		root = tree(t, map[string]string{"real.txt": "public line\n" + secret + "\nthird\nfourth\n"})
		if err := os.Symlink("real.txt", filepath.Join(root, "link.txt")); err != nil {
			t.Skipf("this filesystem will not create a symlink: %v", err)
		}
		return root, "real.txt", "link.txt"
	}

	t.Run("a partial read does not license the alias spelling", func(t *testing.T) {
		root, real, alias := aliased(t)
		before, err := os.ReadFile(filepath.Join(root, real))
		if err != nil {
			t.Fatal(err)
		}
		observed, _ := read.Run(io.Discard, root,
			[]read.Spec{{Path: real, Ranges: []read.Range{{Start: 1, End: 1}}}}, read.Options{})

		res, err := apply.Apply(root, []apply.Input{{
			Path: alias, Start: 4, End: 4, Op: "replace",
			Body: []string{"PWNED"}, Lines: unset,
		}}, apply.Options{Seen: observed})
		if err != nil {
			t.Fatal(err)
		}
		if res.Failed != 1 {
			t.Fatalf("failed=%d, want 1 — line 4 was never served under any spelling", res.Failed)
		}
		if !strings.Contains(res.Hunks[0].Reason, "has not been read") {
			t.Errorf("the refusal is not the ledger's: %s", res.Hunks[0].Reason)
		}
		after, err := os.ReadFile(filepath.Join(root, real))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(before, after) {
			t.Fatalf("the file changed through the alias spelling:\n%s", after)
		}
	})

	// Issue #47's promise, and the reason this fix is a resolution rather than a
	// ban: a file that HAS been read must not be refused as unread because the
	// caller typed another valid name for it.
	t.Run("a whole read still licenses the alias spelling", func(t *testing.T) {
		root, real, alias := aliased(t)
		observed, _ := read.Run(io.Discard, root,
			[]read.Spec{{Path: real}}, read.Options{})

		res, err := apply.Apply(root, []apply.Input{{
			Path: alias, Start: 4, End: 4, Op: "replace",
			Body: []string{"rewritten"}, Lines: unset,
		}}, apply.Options{Seen: observed})
		if err != nil {
			t.Fatal(err)
		}
		if res.Failed != 0 {
			t.Fatalf("failed=%d, want 0 — the whole file was served, and #47 says the spelling may differ: %s",
				res.Failed, res.Hunks[0].Reason)
		}
	})

	// The anchored case, which is ADR-028's property reaching the alias: with the
	// per-line gate absent, a failed anchor printed the line as it always had.
	t.Run("an alias-spelled anchor failure reads back no unserved line", func(t *testing.T) {
		root, real, alias := aliased(t)
		observed, _ := read.Run(io.Discard, root,
			[]read.Spec{{Path: real, Ranges: []read.Range{{Start: 1, End: 1}}}}, read.Options{})

		res, err := apply.Apply(root, []apply.Input{{
			Path: alias, Start: 2, End: 2, Op: "replace",
			Body: []string{"rewritten"}, Lines: unset, Anchor: "no-such-text",
		}}, apply.Options{Seen: observed})
		if err != nil {
			t.Fatal(err)
		}
		if res.Failed != 1 {
			t.Fatalf("failed=%d, want 1", res.Failed)
		}
		if strings.Contains(res.Hunks[0].Reason, secret) {
			t.Errorf("the refusal reads back a line the caller was never served: %s", res.Hunks[0].Reason)
		}
	})

	// The case-only alias needs a case-INsensitive filesystem, so it runs on
	// Windows CI and on a developer's macOS and cannot run on Linux, where
	// real.txt and REAL.txt are two different files. Probed, never asserted.
	t.Run("a case-only alias is the same file too", func(t *testing.T) {
		root := tree(t, map[string]string{"real.txt": "public line\n" + secret + "\nthird\nfourth\n"})
		if _, err := os.Stat(filepath.Join(root, "REAL.TXT")); err != nil {
			t.Skip("this filesystem is case-sensitive: a case-only alias names a different file")
		}
		observed, _ := read.Run(io.Discard, root,
			[]read.Spec{{Path: "real.txt", Ranges: []read.Range{{Start: 1, End: 1}}}}, read.Options{})

		res, err := apply.Apply(root, []apply.Input{{
			Path: "REAL.TXT", Start: 4, End: 4, Op: "replace",
			Body: []string{"PWNED"}, Lines: unset,
		}}, apply.Options{Seen: observed})
		if err != nil {
			t.Fatal(err)
		}
		if res.Failed != 1 {
			t.Fatalf("failed=%d, want 1 — line 4 was never served under any spelling", res.Failed)
		}
	})
}
