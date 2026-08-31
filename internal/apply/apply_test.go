package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	lines, nl, _, err := readLines(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	return shaOf(lines, nl)
}

// Read before modify: a range address only means something in the version of
// the file those line numbers were counted in, so editing a file mrw has never
// seen is writing against a picture that may already be wrong.
func TestAFileNeverSeenCannotBeEdited(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "f.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1, Index: 0},
	}, Options{Seen: map[string]string{}})
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
	}, Options{Seen: map[string]string{"f.txt": recorded}})
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
	}, Options{Seen: map[string]string{"f.txt": shaOfFile(t, root, "f.txt")}})
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
	}, Options{Seen: map[string]string{}})
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
	}, Options{Seen: map[string]string{}, Force: true})
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
