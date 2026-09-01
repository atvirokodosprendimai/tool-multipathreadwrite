package apply

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func write(t *testing.T, root, name, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, root, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

const abcde = "a\nb\nc\nd\ne\n"

// The headline case from the brief: replace 3-6 and insert after 2, in one
// pass, with both addresses resolved against the ORIGINAL file.
func TestAddressesResolveAgainstTheOriginalFile(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", "1\n2\n3\n4\n5\n6\n7\n")

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 3, End: 6, Op: "replace", Body: []string{"NEW"}, Lines: -1, Index: 0},
		{Path: "f.txt", Start: 2, End: 2, Op: "insert-after", Body: []string{"INS"}, Lines: -1, Index: 1},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Fatalf("failed=%d: %+v", res.Failed, res.Hunks)
	}
	if got, want := read(t, root, "f.txt"), "1\n2\nINS\nNEW\n7\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// The property that removes the classic edit-shifting problem: with sed or a
// hand-rolled script, inserting at line 10 pushes line 100 down, so edits must
// be applied bottom-to-top or every later address is wrong. Here every address
// names the ORIGINAL file, so a plan is written top-to-bottom and no hunk cares
// what the ones before it did — including a delete ABOVE a later insert, which
// is the case that actually shifts things.
func TestEarlierHunksNeverShiftLaterAddresses(t *testing.T) {
	root := t.TempDir()
	var sb strings.Builder
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&sb, "line%02d\n", i)
	}
	write(t, root, "f.txt", sb.String())

	// Deliberately ascending, with a delete between two inserts.
	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 2, End: 2, Op: "insert-after", Body: []string{"A"}, Lines: -1, Index: 0},
		{Path: "f.txt", Start: 5, End: 6, Op: "delete", Lines: -1, Index: 1},
		{Path: "f.txt", Start: 10, End: 10, Op: "insert-after", Body: []string{"B"}, Lines: -1, Index: 2},
		{Path: "f.txt", Start: 15, End: 15, Op: "replace", Body: []string{"C"}, Anchor: "line15", Lines: -1, Index: 3},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Fatalf("%+v", res.Hunks)
	}

	got := strings.Split(strings.TrimSuffix(read(t, root, "f.txt"), "\n"), "\n")
	// Each insert must sit directly after the ORIGINAL line it named, even
	// though two lines were removed above the later ones.
	for _, tc := range []struct{ after, want string }{
		{"line02", "A"},
		{"line10", "B"},
	} {
		i := indexOf(got, tc.after)
		if i < 0 || i+1 >= len(got) || got[i+1] != tc.want {
			t.Errorf("%q is not directly after %q:\n%v", tc.want, tc.after, got)
		}
	}
	if indexOf(got, "line05") >= 0 || indexOf(got, "line06") >= 0 {
		t.Errorf("the delete did not remove original lines 5-6:\n%v", got)
	}
	if indexOf(got, "line15") >= 0 || indexOf(got, "C") < 0 {
		t.Errorf("the replace did not land on original line 15:\n%v", got)
	}
	if len(got) != 20+2-2-1+1 { // 20 original, +2 inserted, -2 deleted, 1 replaced by 1
		t.Errorf("got %d lines, want 20:\n%v", len(got), got)
	}
}

func indexOf(lines []string, want string) int {
	for i, l := range lines {
		if l == want {
			return i
		}
	}
	return -1
}

func TestMultipleFilesInOneRun(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", abcde)
	write(t, root, "sub/b.txt", "x\ny\n")

	res, err := Apply(root, []Input{
		{Path: "a.txt", Start: 2, End: 3, Op: "delete", Lines: -1, Index: 0},
		{Path: "sub/b.txt", Start: 0, End: 0, Op: "insert-after", Body: []string{"top"}, Lines: -1, Index: 1},
		{Path: "sub/c.txt", Op: "create", Body: []string{"made"}, Lines: -1, Index: 2},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Fatalf("failed: %+v", res.Hunks)
	}
	if got := read(t, root, "a.txt"); got != "a\nd\ne\n" {
		t.Errorf("a.txt = %q", got)
	}
	if got := read(t, root, "sub/b.txt"); got != "top\nx\ny\n" {
		t.Errorf("b.txt = %q", got)
	}
	if got := read(t, root, "sub/c.txt"); got != "made\n" {
		t.Errorf("c.txt = %q", got)
	}
	if len(res.Files) != 3 {
		t.Errorf("got %d file results, want 3", len(res.Files))
	}
}

// The defect this whole tool exists to prevent: one hunk of several silently
// matching nothing while the run reports success.
func TestOneBadHunkAbortsEverythingAndSaysWhich(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", abcde)
	write(t, root, "b.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, Index: 0},
		{Path: "a.txt", Start: 2, End: 2, Op: "replace", Body: []string{"B"}, Anchor: "zzz", Lines: -1, Index: 1},
		{Path: "b.txt", Start: 1, End: 1, Op: "replace", Body: []string{"C"}, Lines: -1, Index: 2},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Fatalf("failed=%d, want 1", res.Failed)
	}
	if res.Applied {
		t.Error("Applied is true after a failure")
	}
	if s := res.Hunks[1].Status; s != StatusFailed {
		t.Errorf("hunk 1 status = %s, want failed", s)
	}
	if !strings.Contains(res.Hunks[1].Reason, "anchor") {
		t.Errorf("reason does not name the anchor: %q", res.Hunks[1].Reason)
	}
	// Both the sibling hunk and the untouched second file must be reported as
	// skipped, not ok — "ok but not written" is the lie we are avoiding.
	if s := res.Hunks[0].Status; s != StatusSkipped {
		t.Errorf("hunk 0 status = %s, want skipped", s)
	}
	if s := res.Hunks[2].Status; s != StatusSkipped {
		t.Errorf("hunk 2 status = %s, want skipped", s)
	}
	if read(t, root, "a.txt") != abcde || read(t, root, "b.txt") != abcde {
		t.Error("a file was written despite the failure")
	}
}

// The receipt must name every file the plan ADDRESSED, not only the ones that
// survived validation. A file whose hunk failed used to vanish from Files
// entirely — so a two-file plan reported "1 file(s)" and --json's files[] left
// the failing one out, under-reporting how much the run was about to touch.
func TestFailedFilesStillAppearInTheReceipt(t *testing.T) {
	root := t.TempDir()
	write(t, root, "good.txt", abcde)
	write(t, root, "bad.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "good.txt", Start: 1, End: 1, Op: "replace", Body: []string{"G"}, Lines: -1, Index: 0},
		{Path: "bad.txt", Start: 1, End: 1, Op: "replace", Body: []string{"B"}, Anchor: "zzz", Lines: -1, Index: 1},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Fatalf("failed=%d, want 1", res.Failed)
	}
	if len(res.Files) != 2 {
		t.Fatalf("got %d file result(s), want 2 — every addressed file must be reported: %+v", len(res.Files), res.Files)
	}
	// Plan order, so the receipt reads in the order the caller wrote it.
	if res.Files[0].Path != "good.txt" || res.Files[1].Path != "bad.txt" {
		t.Errorf("files out of plan order: %q, %q", res.Files[0].Path, res.Files[1].Path)
	}
	for _, f := range res.Files {
		if f.Written {
			t.Errorf("%s reported as written after a failed run", f.Path)
		}
	}
	// The failing file still carries what was observed about it, so a caller
	// can tell "could not validate" from "was never looked at".
	if res.Files[1].SHABefore == "" || res.Files[1].LinesFrom != 5 {
		t.Errorf("the failing file lost its observed state: %+v", res.Files[1])
	}
}

// A file addressed by several hunks, one of which fails, is reported once.
func TestAFailedFileIsReportedOnceNotPerHunk(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, Index: 0},
		{Path: "f.txt", Start: 2, End: 2, Op: "replace", Body: []string{"B"}, Anchor: "zzz", Lines: -1, Index: 1},
		{Path: "f.txt", Start: 3, End: 3, Op: "replace", Body: []string{"C"}, Anchor: "qqq", Lines: -1, Index: 2},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Files) != 1 {
		t.Errorf("got %d file result(s), want 1: %+v", len(res.Files), res.Files)
	}
	if res.Failed != 2 {
		t.Errorf("failed=%d, want 2 — every bad hunk is reported, not just the first", res.Failed)
	}
}

// shaOfFile is what the ledger would hold for a file on disk.
func shaOfFile(t *testing.T, root, name string) string {
	t.Helper()
	t2, _, err := readLines(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	return shaOf(t2)
}

// Read before modify: a range address only means something in the version of
// the file those line numbers were counted in, so editing a file mrw has never
// seen is writing against a picture that may already be wrong.
func TestAFileNeverSeenCannotBeEdited(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1, Index: 0},
	}, Options{Seen: map[string]Seen{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Fatalf("an unseen file was edited: %+v", res.Hunks)
	}
	if !strings.Contains(res.Hunks[0].Reason, "has not been read") {
		t.Errorf("reason = %q", res.Hunks[0].Reason)
	}
	if read(t, root, "f.txt") != abcde {
		t.Error("the file was written")
	}
}

// The case the ledger exists for: mrw saw the file, something else changed it,
// and the caller's line numbers now point somewhere else in a file they have
// not looked at.
func TestAFileChangedBehindMrwsBackCannotBeEdited(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)
	recorded := shaOfFile(t, root, "f.txt")

	write(t, root, "f.txt", "totally\ndifferent\n") // someone else edits it

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1, Index: 0},
	}, Options{Seen: map[string]Seen{"f.txt": {SHA: recorded}}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Fatalf("a changed file was edited: %+v", res.Hunks)
	}
	if !strings.Contains(res.Hunks[0].Reason, "changed since") {
		t.Errorf("reason = %q", res.Hunks[0].Reason)
	}
	if read(t, root, "f.txt") != "totally\ndifferent\n" {
		t.Error("the file was written")
	}
}

func TestASeenFileIsEditable(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1, Index: 0},
	}, Options{Seen: map[string]Seen{"f.txt": {SHA: shaOfFile(t, root, "f.txt")}}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Fatalf("%+v", res.Hunks)
	}
	if got := read(t, root, "f.txt"); got != "X\nb\nc\nd\ne\n" {
		t.Errorf("got %q", got)
	}
}

// create is exempt: there is no existing content to have a stale picture of.
func TestCreateNeedsNoPriorObservation(t *testing.T) {
	root := t.TempDir()
	res, err := Apply(root, []Input{
		{Path: "new.txt", Op: "create", Body: []string{"hello"}, Lines: -1, Index: 0},
	}, Options{Seen: map[string]Seen{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Fatalf("create was blocked by the guard: %+v", res.Hunks)
	}
	if read(t, root, "new.txt") != "hello\n" {
		t.Error("the file was not created")
	}
}

func TestForceBypassesTheGuard(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1, Index: 0},
	}, Options{Seen: map[string]Seen{}, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Fatalf("--force did not bypass the guard: %+v", res.Hunks)
	}
}

// A nil ledger means the caller is not using the guard at all — distinct from
// an empty one, which means "I have seen nothing".
func TestNilLedgerDisablesTheGuard(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1, Index: 0},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Fatalf("a nil ledger enforced the guard: %+v", res.Hunks)
	}
}

func TestPreconditions(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	for name, in := range map[string]Input{
		"stale sha":     {Path: "f.txt", Start: 1, End: 1, Op: "delete", SHA: "00000000", Lines: -1},
		"wrong count":   {Path: "f.txt", Start: 1, End: 2, Op: "delete", Lines: 3},
		"bad anchor":    {Path: "f.txt", Start: 1, End: 1, Op: "delete", Anchor: "nope", Lines: -1},
		"past end":      {Path: "f.txt", Start: 9, End: 9, Op: "delete", Lines: -1},
		"missing file":  {Path: "gone.txt", Start: 1, End: 1, Op: "delete", Lines: -1},
		"create exists": {Path: "f.txt", Op: "create", Body: []string{"x"}, Lines: -1},
	} {
		res, err := Apply(root, []Input{in}, Options{})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if res.Failed != 1 {
			t.Errorf("%s: failed=%d, want 1 (%+v)", name, res.Failed, res.Hunks)
		}
	}
	if read(t, root, "f.txt") != abcde {
		t.Error("file changed despite every hunk failing")
	}
}

func TestGoodShaAndCountsPass(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	probe, err := Apply(root, []Input{{Path: "f.txt", Start: 1, End: 1, Op: "delete", Lines: -1}}, Options{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	sha := probe.Files[0].SHABefore

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 2, End: 3, Op: "replace", Body: []string{"X"}, SHA: sha[:12], Lines: 2, Anchor: "b", Index: 0},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Fatalf("failed: %+v", res.Hunks)
	}
	if got := read(t, root, "f.txt"); got != "a\nX\nd\ne\n" {
		t.Errorf("got %q", got)
	}
	if res.Hunks[0].Removed != 2 || res.Hunks[0].Added != 1 {
		t.Errorf("counts = -%d +%d, want -2 +1", res.Hunks[0].Removed, res.Hunks[0].Added)
	}
}

func TestOverlappingHunksAreRejected(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 1, End: 3, Op: "replace", Body: []string{"X"}, Lines: -1, Index: 0},
		{Path: "f.txt", Start: 2, End: 4, Op: "replace", Body: []string{"Y"}, Lines: -1, Index: 1},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed == 0 {
		t.Fatal("overlapping hunks were accepted")
	}
	if !strings.Contains(res.Hunks[1].Reason, "overlap") {
		t.Errorf("reason = %q, want it to name the overlap", res.Hunks[1].Reason)
	}
	if read(t, root, "f.txt") != abcde {
		t.Error("file was written")
	}
}

func TestDryRunWritesNothingButReportsTheOutcome(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 1, End: 5, Op: "delete", Lines: -1, Index: 0},
	}, Options{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 || res.Applied {
		t.Fatalf("res = %+v", res)
	}
	if res.Files[0].Written {
		t.Error("dry run reported a write")
	}
	if res.Files[0].LinesTo != 0 || res.Files[0].SHAAfter == res.Files[0].SHABefore {
		t.Errorf("dry run did not compute the result: %+v", res.Files[0])
	}
	if read(t, root, "f.txt") != abcde {
		t.Error("dry run wrote the file")
	}
}

// EOF addressing lets a plan target the end of a file whose length the caller
// never had to read.
func TestEOFAddressing(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: EOF, End: EOF, Op: "insert-after", Body: []string{"last"}, Lines: -1, Index: 0},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Fatalf("%+v", res.Hunks)
	}
	if got := read(t, root, "f.txt"); got != "a\nb\nc\nd\ne\nlast\n" {
		t.Errorf("got %q", got)
	}
}

// A file without a trailing newline must not silently gain one.
func TestTrailingNewlineIsPreserved(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", "a\nb")

	if _, err := Apply(root, []Input{
		{Path: "f.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, Index: 0},
	}, Options{}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, root, "f.txt"); got != "A\nb" {
		t.Errorf("got %q, want %q", got, "A\nb")
	}
}

func TestFilePermissionsSurvive(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.sh", "a\n")
	if err := os.Chmod(filepath.Join(root, "f.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(root, []Input{
		{Path: "f.sh", Start: 1, End: 1, Op: "replace", Body: []string{"b"}, Lines: -1, Index: 0},
	}, Options{}); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(root, "f.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o755 {
		t.Errorf("perm = %v, want 0755", fi.Mode().Perm())
	}
}

// ADR-008 T1. A bodyless `delete` consumes a range while asserting nothing
// about it, and its receipt reported only a count. The bounds are what make a
// wrong range visible at write time: the incident that produced the ADR was a
// 4-line delete that took a closing brace and reported `-4 +0  ok`.
func TestDeleteRecordsItsBounds(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", "a\nb\nc\nd\ne\n")

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 2, End: 4, Op: "delete", Lines: -1, Index: 0},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Fatalf("failed=%d: %+v", res.Failed, res.Hunks)
	}
	if got, want := res.Hunks[0].RemovedFirst, "b"; got != want {
		t.Errorf("RemovedFirst = %q, want %q", got, want)
	}
	if got, want := res.Hunks[0].RemovedLast, "d"; got != want {
		t.Errorf("RemovedLast = %q, want %q", got, want)
	}
}

// The bounds go through the same `trim` the anchor-failure message uses, so one
// convention covers both: leading and trailing space gone, 60 characters at
// most. A receipt line that wraps is a receipt line a reader skips.
func TestDeleteBoundsAreTrimmed(t *testing.T) {
	root := t.TempDir()
	long := strings.Repeat("x", 80)
	write(t, root, "f.txt", "\t  indented  \n"+long+"\n")

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 1, End: 2, Op: "delete", Lines: -1, Index: 0},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := res.Hunks[0].RemovedFirst, "indented"; got != want {
		t.Errorf("RemovedFirst = %q, want %q", got, want)
	}
	if got, want := res.Hunks[0].RemovedLast, trim(long); got != want {
		t.Errorf("RemovedLast = %q, want %q", got, want)
	}
	if len(res.Hunks[0].RemovedLast) > 60 {
		t.Errorf("RemovedLast is %d characters; the receipt must stay bounded", len(res.Hunks[0].RemovedLast))
	}
}

// The degenerate range: one line removed is both the first and the last, and
// saying so twice is better than a special case the reader has to learn.
func TestAOneLineDeleteRecordsTheSameLineTwice(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 3, End: 3, Op: "delete", Lines: -1, Index: 0},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Hunks[0].RemovedFirst != "c" || res.Hunks[0].RemovedLast != "c" {
		t.Errorf("got from %q to %q, want both %q", res.Hunks[0].RemovedFirst, res.Hunks[0].RemovedLast, "c")
	}
}

// No other op gains fields. `replace` and the insertions already carry a body
// the caller wrote, and the two are not equally strong: writing a `replace`
// body means reading the lines it replaces, while an insertion's body says
// what to add and nothing about WHERE — `anchor=` is what pins that. Only a
// `delete` body is machine-checked against the range it addresses. Either way
// a bounds field here would be noise: an empty `from "" to ""` on every other
// receipt line says nothing.
func TestOnlyDeleteRecordsBounds(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, Index: 0},
		{Path: "f.txt", Start: 2, End: 2, Op: "insert-after", Body: []string{"INS"}, Lines: -1, Index: 1},
		{Path: "f.txt", Start: 3, End: 3, Op: "insert-before", Body: []string{"PRE"}, Lines: -1, Index: 2},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Fatalf("failed=%d: %+v", res.Failed, res.Hunks)
	}
	for _, h := range res.Hunks {
		if h.RemovedFirst != "" || h.RemovedLast != "" {
			t.Errorf("%s recorded bounds %q..%q; only delete records them", h.Op, h.RemovedFirst, h.RemovedLast)
		}
		// Emptiness is not enough. HunkResult is one struct for all five ops,
		// so without `omitempty` a replace would carry `"removed_first": ""`
		// and still satisfy the check above — while breaking the contract,
		// which promises these fields ON A DELETE HUNK. Assert ABSENCE.
		b, err := json.Marshal(h)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "removed_first") || strings.Contains(string(b), "removed_last") {
			t.Errorf("%s carries the delete-only fields in --json: %s", h.Op, b)
		}
	}
}

// ADR-008 T2. The body of a delete is the caller's expectation of what the
// range holds. When it matches, the delete applies exactly as a bodyless one.
func TestADeleteWhoseExpectedRemovalMatchesApplies(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 2, End: 3, Op: "delete", Body: []string{"b", "c"}, Lines: -1, Index: 0},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Fatalf("a matching expected removal was rejected: %+v", res.Hunks)
	}
	if got, want := read(t, root, "f.txt"), "a\nd\ne\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// The comparison is line for line, and a mismatch says WHICH line differed —
// the same shape as the anchor failure, which prints the anchor beside the real
// line. A refusal that does not say where it disagreed is barely better than no
// guard at all.
func TestADeleteWhoseExpectedRemovalDiffersNamesTheLine(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 2, End: 3, Op: "delete", Body: []string{"b", "NOT-C"}, Lines: -1, Index: 0},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Fatalf("a mismatched expected removal was accepted: %+v", res.Hunks)
	}
	reason := res.Hunks[0].Reason
	if !strings.Contains(reason, "3") || !strings.Contains(reason, "NOT-C") || !strings.Contains(reason, "c") {
		t.Errorf("the refusal does not name the line, the expectation and what is there: %q", reason)
	}
	if got := read(t, root, "f.txt"); got != abcde {
		t.Errorf("the file changed: %q", got)
	}
}

// A body of the wrong LENGTH is the miscounted range this whole record is
// about, so it must fail rather than compare only as far as the shorter of the
// two.
func TestAnExpectedRemovalOfTheWrongLengthIsRejected(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 2, End: 4, Op: "delete", Body: []string{"b", "c"}, Lines: -1, Index: 0},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Fatalf("a 2-line expectation was accepted for a 3-line range: %+v", res.Hunks)
	}
	if got := read(t, root, "f.txt"); got != abcde {
		t.Errorf("the file changed: %q", got)
	}
}

// The opt-in stays opt-in: a delete with no body behaves exactly as it did
// before this task, guards and all.
func TestABodylessDeleteIsUnchanged(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 2, End: 3, Op: "delete", Lines: -1, Index: 0},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Fatalf("a bodyless delete failed: %+v", res.Hunks)
	}
	if got, want := read(t, root, "f.txt"), "a\nd\ne\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// The expected-removal comparison quotes the line the file actually holds, so
// it must run BELOW covered(): a caller served lines 3-3 who addresses line 1
// would otherwise get line 1 read back inside a refusal, which is exactly the
// disclosure ADR-002 and ADR-005 draw the boundary around.
//
// The fixture is the whole point, and the first version of this test got it
// wrong: it passed an EMPTY Seen map, so the file was not `known` and the
// WHOLE-FILE gate answered ~130 lines earlier — the body comparison was never
// reached in either arrangement, and the test passed with the ordering
// reversed. A reviewer proved that by putting the defect back and watching the
// whole suite stay green. Here the file IS known and its sha matches; only the
// addressed RANGE was never served, so covered() is the gate under test.
func TestAnExpectedRemovalIsNotCheckedAgainstAnUnseenFile(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 1, End: 1, Op: "delete", Body: []string{"GUESS"}, Lines: -1, Index: 0},
	}, Options{Seen: map[string]Seen{
		// Served line 3 and nothing else. The hunk addresses line 1.
		"f.txt": {SHA: shaOfFile(t, root, "f.txt"), Spans: [][2]int{{3, 3}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Fatalf("a hunk addressing an unserved line was accepted: %+v", res.Hunks)
	}
	reason := res.Hunks[0].Reason
	if !strings.Contains(reason, "has not been read") {
		t.Errorf("reason = %q, want the range-level ledger refusal", reason)
	}
	if strings.Contains(reason, "expected removal") {
		t.Errorf("the body was compared before the ledger check: %q", reason)
	}
	// The line itself must not appear. abcde's line 1 is "a", too short to
	// search for, so assert on the shape instead: a refusal naming the
	// comparison is the only way that line could have reached the caller.
	if strings.Contains(reason, "GUESS") {
		t.Errorf("the refusal quoted the unserved line back: %q", reason)
	}
}

// The commonest mismatch a hand-written expected removal produces is
// whitespace: a tab where the caller typed spaces, a dropped trailing space, a
// stray CR. Trimming both sides before printing them made the refusal say
// `plan says "indented", file has "indented"` — two identical strings and no
// way to act on it, which is the outcome T2's Stop Condition names as worse
// than having no guard.
func TestAnExpectedRemovalDiffersOnlyInWhitespace(t *testing.T) {
	for _, c := range []struct{ name, file, body string }{
		{"tab against spaces", "\tindented", "    indented"},
		{"dropped trailing space", "beta ", "beta"},
		{"stray carriage return", "x\r", "x"},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			write(t, root, "f.txt", "a\n"+c.file+"\nc\n")

			res, err := Apply(root, []Input{
				{Path: "f.txt", Start: 2, End: 2, Op: "delete", Body: []string{c.body}, Lines: -1, Index: 0},
			}, Options{})
			if err != nil {
				t.Fatal(err)
			}
			if res.Failed != 1 {
				t.Fatalf("a whitespace mismatch was accepted: %+v", res.Hunks)
			}
			// The two halves of the message must not be the same string. %q
			// renders a tab as \t and holds a trailing space inside the
			// quotes, so the difference survives — as long as nothing trimmed
			// it away first.
			reason := res.Hunks[0].Reason
			plan, file, found := strings.Cut(reason, ", file has ")
			if !found {
				t.Fatalf("unexpected message shape: %q", reason)
			}
			_, plan, _ = strings.Cut(plan, "plan says ")
			if plan == file {
				t.Errorf("the refusal shows two identical strings: %q", reason)
			}
		})
	}
}

// trim used to slice at 57 BYTES, which splits a multi-byte rune. ADR-008 put
// its output into --json, where an invalid UTF-8 fragment is re-encoded as
// U+FFFD — a silently corrupted machine-readable field.
func TestDeleteBoundsAreTrimmedOnARuneBoundary(t *testing.T) {
	root := t.TempDir()
	long := strings.Repeat("x", 55) + "😀😀😀"
	write(t, root, "f.txt", long+"\nb\n")

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 1, End: 2, Op: "delete", Lines: -1, Index: 0},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	got := res.Hunks[0].RemovedFirst
	if !utf8.ValidString(got) {
		t.Errorf("RemovedFirst is not valid UTF-8: %q", got)
	}
	if strings.ContainsRune(got, utf8.RuneError) {
		t.Errorf("RemovedFirst carries a replacement rune: %q", got)
	}
	if len(got) > 60 {
		t.Errorf("RemovedFirst is %d bytes; the receipt must stay bounded", len(got))
	}
}

// A delete of blank lines removed something, and both surfaces must say so.
// `omitempty` keys on the VALUE, so it dropped the fields for exactly this
// delete while the human receipt line — keyed on the op — printed
// `from "" to ""`. Presence in --json is the op, not the value.
func TestADeleteOfBlankLinesStillRecordsBounds(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", "a\n\n\nb\n")

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 2, End: 3, Op: "delete", Lines: -1, Index: 0},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Fatalf("failed=%d: %+v", res.Failed, res.Hunks)
	}
	b, err := json.Marshal(res.Hunks[0])
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(b, &wire); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"removed_first", "removed_last"} {
		v, present := wire[k]
		if !present {
			t.Errorf("%s is absent on a delete hunk: %s", k, b)
			continue
		}
		if v != "" {
			t.Errorf("%s = %v, want the empty line it removed", k, v)
		}
	}
}

// An ABSOLUTE path in a plan is joined to the root, so it names something under
// the root that is almost never there. Reporting a bare "does not exist" sent
// the caller looking for a file they can see with their own eyes; the message
// has to say the path was reinterpreted.
func TestAnAbsolutePathSaysItWasResolvedAgainstTheRoot(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "/etc/hosts", Start: 1, End: 1, Op: "delete", Lines: -1, Index: 0},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Fatalf("an absolute path was accepted: %+v", res.Hunks)
	}
	if reason := res.Hunks[0].Reason; !strings.Contains(reason, "absolute") ||
		!strings.Contains(reason, "relative to the root") {
		t.Errorf("reason = %q, want it to say the path was resolved against the root", reason)
	}
}

// `$` resolves in this package, so a reversed range built from it escapes the
// parser's own check and used to be reported as "out of range" — which it is
// not. `$-1` on a 5-line file is 5-1: the ends are the wrong way round.
func TestAReversedRangeFromEOFSaysSo(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: EOF, End: 1, Op: "delete", Lines: -1, Index: 0},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Fatalf("a reversed range was accepted: %+v", res.Hunks)
	}
	reason := res.Hunks[0].Reason
	if !strings.Contains(reason, "ends before it starts") {
		t.Errorf("reason = %q, want the reversed-range message", reason)
	}
	if strings.Contains(reason, "out of range") {
		t.Errorf("a reversed range is not an out-of-range one: %q", reason)
	}
}

// The bounds are populated in the splice, which runs only for hunks that land.
// Keying --json presence on the op alone therefore marshalled `""` for a failed
// or skipped delete, making it indistinguishable from a successful delete of
// blank lines — the one case the fields exist to make legible.
func TestAFailedOrSkippedDeleteRecordsNoBounds(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 99, End: 99, Op: "replace", Body: []string{"x"}, Lines: -1, Index: 0},
		{Path: "f.txt", Start: 3, End: 3, Op: "delete", Lines: -1, Index: 1},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed == 0 {
		t.Fatalf("expected the out-of-range replace to fail: %+v", res.Hunks)
	}
	for _, h := range res.Hunks {
		if h.Status == StatusOK {
			continue
		}
		b, err := json.Marshal(h)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "removed_first") || strings.Contains(string(b), "removed_last") {
			t.Errorf("a %s %s hunk carries bounds it never recorded: %s", h.Status, h.Op, b)
		}
	}
}

// A plan aborts on a filesystem failure with the tree UNTOUCHED. The write
// phase used to write each file and move on, so a later file that could not be
// written left the earlier ones already renamed into place — a partially
// applied plan, which ADR-001 rule 2 calls worse than no change. Three things
// went wrong at once and each is asserted here: the tree was changed, no
// receipt was produced, and the ledger (recorded from that receipt) then held
// the pre-write hash, so mrw refused the caller's NEXT edit to a file it had
// modified itself.
//
// The trigger is forced through the stageFileFn seam rather than through an
// unwritable directory, because `chmod 0555` is a no-op for uid 0: a
// permission-based test exercises nothing when CI runs as root, and a row that
// cannot fail is exactly the defect this guard is about. scripts/contract.sh
// carries the real-permissions version, skipped when the bits are not enforced.
func TestAFailedStageLeavesTheTreeUntouched(t *testing.T) {
	root := t.TempDir()
	write(t, root, "one.txt", abcde)
	write(t, root, "two.txt", abcde)

	real := stageFileFn
	t.Cleanup(func() { stageFileFn = real })
	calls := 0
	stageFileFn = func(path string, tx text) (string, string, error) {
		calls++
		if calls == 2 {
			return "", "", errors.New("staging refused")
		}
		return real(path, tx)
	}

	res, err := Apply(root, []Input{
		{Path: "one.txt", Start: 1, End: 1, Op: "replace", Body: []string{"CHANGED"}, Lines: -1, Index: 0},
		{Path: "two.txt", Start: 1, End: 1, Op: "replace", Body: []string{"CHANGED"}, Lines: -1, Index: 1},
	}, Options{Force: true})
	if err == nil {
		t.Fatalf("a failed stage was reported as success: %+v", res)
	}
	if res.Applied {
		t.Error("Applied is true after an aborted write")
	}
	// The substance: the FIRST file, staged before the failure, must not have
	// been renamed into place. Asserting only on the error would pass while
	// the tree was half-written, which is the whole defect.
	for _, name := range []string{"one.txt", "two.txt"} {
		got, rerr := os.ReadFile(filepath.Join(root, name))
		if rerr != nil {
			t.Fatal(rerr)
		}
		if strings.Contains(string(got), "CHANGED") {
			t.Errorf("%s was written despite the plan aborting", name)
		}
	}
	// ADR-004: nothing left in the working tree. A staged temp that is never
	// renamed has to be unlinked, or an aborted plan litters .mrw-* beside
	// every target it reached.
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".mrw-") {
			t.Errorf("staging left %s behind", e.Name())
		}
	}
}

// The residual case staging cannot remove: a rename that fails after earlier
// renames succeeded really does leave a partial tree, so the error has to name
// what is already on disk. A caller told only "permission denied" cannot tell
// an untouched tree from a half-written one.
func TestAPartialWriteNamesWhatWasAlreadyWritten(t *testing.T) {
	if got := writtenSoFar(nil); !strings.Contains(got, "nothing was written") {
		t.Errorf("empty case = %q", got)
	}
	got := writtenSoFar([]FileResult{{Path: "a.go"}, {Path: "b.go"}})
	if !strings.Contains(got, "a.go") || !strings.Contains(got, "b.go") {
		t.Errorf("a partial write did not name its files: %q", got)
	}
}
