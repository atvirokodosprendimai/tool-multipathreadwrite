package apply

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// long is one path component no filesystem mrw runs on accepts: 300 bytes is
// past the 255-byte NAME_MAX of Linux, macOS and Windows alike, so it is a
// deterministic trigger that needs no permissions and works as root.
var long = strings.Repeat("x", 300)

func hunkFor(t *testing.T, res Result, path string) HunkResult {
	t.Helper()
	for _, h := range res.Hunks {
		if h.Path == path {
			return h
		}
	}
	t.Fatalf("no hunk for %s in %+v", path, res.Hunks)
	return HunkResult{}
}

// ADR-066 T1. A rename's destination directory used to be created at COMMIT,
// after the plan's other files were already renamed into place, so a
// directory that could not be made left the plan half-applied while every
// hunk read `ok`. The destination here sits under a parent that does not
// exist, so Lstat answers "does not exist", validation passes, and only the
// MkdirAll fails — which staging must now catch before anything is written.
func TestARenameWhoseDestinationDirectoryCannotBeCreatedWritesNothing(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", abcde)
	write(t, root, "b.txt", "bee\n")

	res, err := Apply(root, []Input{
		{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"CHANGED"}, Lines: -1, Index: 0},
		{Path: "b.txt", Op: "rename", Body: []string{"n/" + long + "/b.txt"}, Lines: -1, Index: 1},
	}, Options{})
	if err == nil {
		t.Fatalf("an uncreatable destination directory was not reported: %+v", res)
	}
	if got := read(t, root, "a.txt"); got != abcde {
		t.Fatalf("a.txt was written by a plan that could not rename: %q", got)
	}
	if got := read(t, root, "b.txt"); got != "bee\n" {
		t.Fatalf("b.txt = %q, want it untouched", got)
	}
	if _, err := os.Stat(filepath.Join(root, "n")); !os.IsNotExist(err) {
		t.Fatalf("the aborted plan left n/ behind: %v", err)
	}
	if h := hunkFor(t, res, "b.txt"); h.Status != StatusFailed {
		t.Fatalf("rename status %s, want failed (reason %q)", h.Status, h.Reason)
	}
	if h := hunkFor(t, res, "a.txt"); h.Status != StatusSkipped {
		t.Fatalf("edit status %s, want skipped", h.Status)
	}
	if res.Failed != 1 || res.Applied {
		t.Fatalf("Failed=%d Applied=%v, want 1 and false", res.Failed, res.Applied)
	}
}

// ADR-066 T1. A destination name the filesystem rejects (here ENAMETOOLONG on
// the leaf, under a directory that exists) used to pass validation because
// only `err == nil` was acted on, and then fail at the commit rename. It is a
// fact about the plan, knowable before any write, so it is a failed hunk:
// exit 1, nothing written, no filesystem error returned.
func TestARenameWhoseDestinationNameIsRejectedFailsValidation(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", abcde)
	write(t, root, "b.txt", "bee\n")
	if err := os.Mkdir(filepath.Join(root, "ok"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Apply(root, []Input{
		{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"CHANGED"}, Lines: -1, Index: 0},
		{Path: "b.txt", Op: "rename", Body: []string{"ok/" + long}, Lines: -1, Index: 1},
	}, Options{})
	if err != nil {
		t.Fatalf("a rejected name reached the filesystem: %v", err)
	}
	h := hunkFor(t, res, "b.txt")
	if h.Status != StatusFailed || h.Reason == "" {
		t.Fatalf("rename status %s reason %q, want failed with the Lstat error", h.Status, h.Reason)
	}
	if got := read(t, root, "a.txt"); got != abcde {
		t.Fatalf("a.txt was written: %q", got)
	}
	if got := read(t, root, "b.txt"); got != "bee\n" {
		t.Fatalf("b.txt = %q, want it untouched", got)
	}
}

// ADR-066 T1. Directories staging made for one rename are taken back when a
// LATER rename cannot stage (ADR-004). The control run proves staging is what
// makes new/deep, so the absence asserted afterwards is a removal and not a
// directory that was never created.
func TestAStagedRenameDirectoryIsTakenBackWhenALaterRenameCannotStage(t *testing.T) {
	control := t.TempDir()
	write(t, control, "c.txt", "see\n")
	if _, err := Apply(control, []Input{
		{Path: "c.txt", Op: "rename", Body: []string{"new/deep/c.txt"}, Lines: -1, Index: 0},
	}, Options{}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, control, "new/deep/c.txt"); got != "see\n" {
		t.Fatalf("control rename did not land: %q", got)
	}

	root := t.TempDir()
	write(t, root, "c.txt", "see\n")
	write(t, root, "d.txt", "dee\n")
	res, err := Apply(root, []Input{
		{Path: "c.txt", Op: "rename", Body: []string{"new/deep/c.txt"}, Lines: -1, Index: 0},
		{Path: "d.txt", Op: "rename", Body: []string{"m/" + long + "/d.txt"}, Lines: -1, Index: 1},
	}, Options{})
	if err == nil {
		t.Fatalf("an uncreatable destination directory was not reported: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(root, "new")); !os.IsNotExist(err) {
		t.Fatalf("the aborted plan left new/ behind: %v", err)
	}
	for _, name := range []string{"c.txt", "d.txt"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatalf("%s is gone after an aborted plan: %v", name, err)
		}
	}
}

// failRenames swaps commitRenameFn for one that refuses the renames named by
// match(oldpath, newpath), and restores it when the test ends.
func failRenames(t *testing.T, match func(oldpath, newpath string) bool) {
	t.Helper()
	real := commitRenameFn
	t.Cleanup(func() { commitRenameFn = real })
	commitRenameFn = func(oldpath, newpath string) error {
		if match(oldpath, newpath) {
			return errors.New("injected rename failure")
		}
		return real(oldpath, newpath)
	}
}

// replacingRenamePlan unlinks c.txt, renames b.txt onto c.txt, then renames
// d.txt to e.txt — the shape that lost data on v1.22.3 when the last rename
// failed: restore() moved the unlinked c.txt back over the file just renamed
// onto it, and B-CONTENT existed nowhere afterwards.
func replacingRenamePlan(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "c.txt", "OLD-C\n")
	write(t, root, "b.txt", "B-CONTENT\n")
	write(t, root, "d.txt", "D\n")
	return root
}

func replacingRenameInputs() []Input {
	return []Input{
		{Path: "c.txt", Op: "unlink", Lines: -1, Index: 0},
		{Path: "b.txt", Op: "rename", Body: []string{"c.txt"}, Lines: -1, Index: 1},
		{Path: "d.txt", Op: "rename", Body: []string{"e.txt"}, Lines: -1, Index: 2},
	}
}

func asideFiles(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".mrw-aside-") {
			out = append(out, e.Name())
		}
	}
	return out
}

// ADR-066 T2. A failure while committing unlinks and renames undoes every
// rename that completed, newest first, and only then restores the unlinked
// files — so the file renamed onto an unlinked path goes back to its source
// before the unlinked file returns to that path.
func TestAFailedRenameAfterAReplacingRenameLosesNoFile(t *testing.T) {
	root := replacingRenamePlan(t)
	failRenames(t, func(_, newpath string) bool { return filepath.Base(newpath) == "e.txt" })

	_, err := Apply(root, replacingRenameInputs(), Options{})
	if err == nil {
		t.Fatal("a failed rename was not reported")
	}
	for name, want := range map[string]string{"c.txt": "OLD-C\n", "b.txt": "B-CONTENT\n", "d.txt": "D\n"} {
		if got := read(t, root, name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	if a := asideFiles(t, root); len(a) != 0 {
		t.Errorf("the undo left %v behind", a)
	}
}

// ADR-066 T2. When an undo step itself fails, nothing is overwritten: the
// file renamed onto c.txt stays there, the unlinked c.txt stays in its aside
// as a recovery file the error names, and the receipt keeps the rename
// reported as written. Both original contents exist on disk.
func TestAnUndoThatFailsLosesNoFile(t *testing.T) {
	root := replacingRenamePlan(t)
	failRenames(t, func(oldpath, newpath string) bool {
		return filepath.Base(newpath) == "e.txt" ||
			(filepath.Base(oldpath) == "c.txt" && filepath.Base(newpath) == "b.txt")
	})

	res, err := Apply(root, replacingRenameInputs(), Options{})
	if err == nil {
		t.Fatal("a failed rename was not reported")
	}
	if got := read(t, root, "c.txt"); got != "B-CONTENT\n" {
		t.Fatalf("c.txt = %q: the un-undone rename was overwritten", got)
	}
	a := asideFiles(t, root)
	if len(a) != 1 {
		t.Fatalf("aside files %v, want exactly the one holding OLD-C", a)
	}
	if got := read(t, root, a[0]); got != "OLD-C\n" {
		t.Fatalf("the recovery aside holds %q, want OLD-C", got)
	}
	if !strings.Contains(err.Error(), a[0]) {
		t.Errorf("the error does not name the recovery file %s: %v", a[0], err)
	}
	if got := read(t, root, "d.txt"); got != "D\n" {
		t.Errorf("d.txt = %q, want D", got)
	}
	var renamed bool
	for _, f := range res.Files {
		if f.Path == "b.txt" && f.Written && f.Removed {
			renamed = true
		}
	}
	if !renamed {
		t.Errorf("the rename that could not be undone is not reported written: %+v", res.Files)
	}
}

// contentCommitPlan edits three .go files (non-prose, so Balance is computed)
// with a brace-changing single-line replace each, and renames s.txt into a new
// directory, which staging makes before the content commit starts.
func contentCommitPlan(t *testing.T) (string, []Input) {
	t.Helper()
	root := t.TempDir()
	for _, n := range []string{"a.go", "b.go", "c.go"} {
		write(t, root, n, "x\ny\nz\n")
	}
	write(t, root, "s.txt", "ess\n")
	return root, []Input{
		{Path: "a.go", Start: 1, End: 1, Op: "replace", Body: []string{"{"}, Lines: -1, Index: 0},
		{Path: "b.go", Start: 1, End: 1, Op: "replace", Body: []string{"{"}, Lines: -1, Index: 1},
		{Path: "c.go", Start: 1, End: 1, Op: "replace", Body: []string{"{"}, Lines: -1, Index: 2},
		{Path: "s.txt", Op: "rename", Body: []string{"new/r.txt"}, Lines: -1, Index: 3},
	}
}

// ADR-066 T3. A content rename that fails at commit leaves a receipt that says
// what reached disk: the file already renamed into place is ok and written,
// the one that failed is failed, the rest are skipped and carry no write
// detail (Echo, Balance), Advisories counts only ok hunks, and the directory
// staged for the rename that never ran is taken back.
func TestAFailedContentCommitReportsWrittenHunksOkAndTheRestSkipped(t *testing.T) {
	control, in := contentCommitPlan(t)
	ctl, err := Apply(control, in, Options{EchoPad: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"a.go", "b.go", "c.go"} {
		if h := hunkFor(t, ctl, n); h.Balance == "" || len(h.Echo) == 0 {
			t.Fatalf("control: %s carries no Balance/Echo, so this fixture proves nothing: %+v", n, h)
		}
	}

	root, in := contentCommitPlan(t)
	failRenames(t, func(_, newpath string) bool { return filepath.Base(newpath) == "b.go" })
	res, err := Apply(root, in, Options{EchoPad: 1})
	if err == nil {
		t.Fatal("a failed commit was not reported")
	}
	if h := hunkFor(t, res, "a.go"); h.Status != StatusOK || h.Balance == "" {
		t.Errorf("a.go: status %s balance %q, want ok with its Balance", h.Status, h.Balance)
	}
	if h := hunkFor(t, res, "b.go"); h.Status != StatusFailed || h.Reason == "" {
		t.Errorf("b.go: status %s reason %q, want failed with the error", h.Status, h.Reason)
	}
	for _, n := range []string{"b.go", "c.go", "s.txt"} {
		h := hunkFor(t, res, n)
		if n != "b.go" && h.Status != StatusSkipped {
			t.Errorf("%s: status %s, want skipped", n, h.Status)
		}
		if h.Balance != "" || len(h.Echo) != 0 {
			t.Errorf("%s is not ok but still carries write detail: balance %q echo %v", n, h.Balance, h.Echo)
		}
	}
	if res.Failed != 1 || res.Applied || res.Advisories != 1 {
		t.Errorf("Failed=%d Applied=%v Advisories=%d, want 1, false, 1", res.Failed, res.Applied, res.Advisories)
	}
	written := map[string]bool{}
	for _, f := range res.Files {
		written[f.Path] = f.Written
	}
	if !written["a.go"] || written["b.go"] || written["c.go"] {
		t.Errorf("files written = %v, want only a.go", written)
	}
	if _, ok := written["c.go"]; !ok {
		t.Errorf("c.go, addressed but unwritten, is missing from files: %+v", res.Files)
	}
	if _, err := os.Stat(filepath.Join(root, "new")); !os.IsNotExist(err) {
		t.Errorf("the rename directory staged for a rename that never ran is still there: %v", err)
	}
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".mrw-") {
			t.Errorf("the failed commit left %s behind", e.Name())
		}
	}
}

// ADR-066 T3. After T2's undo puts every unlink and rename back, the receipt
// says nothing was written: no path-op hunk is ok, and writtenSoFar does not
// name the undone records.
func TestAFailedPathOpCommitReportsNothingWrittenAfterTheUndo(t *testing.T) {
	root := replacingRenamePlan(t)
	failRenames(t, func(_, newpath string) bool { return filepath.Base(newpath) == "e.txt" })
	res, err := Apply(root, replacingRenameInputs(), Options{})
	if err == nil {
		t.Fatal("a failed rename was not reported")
	}
	for _, h := range res.Hunks {
		if h.Status == StatusOK {
			t.Errorf("%s %s is ok, but the undo put it back", h.Path, h.Op)
		}
	}
	if h := hunkFor(t, res, "d.txt"); h.Status != StatusFailed {
		t.Errorf("d.txt: status %s, want failed", h.Status)
	}
	for _, f := range res.Files {
		if f.Written {
			t.Errorf("%s is recorded written after the undo", f.Path)
		}
	}
	if !strings.Contains(err.Error(), "nothing was written") {
		t.Errorf("the error does not say nothing was written: %v", err)
	}
}

// ADR-066 T1 (Codex review of #207). A dangling symlink the run did not make
// survives an abort. missingDirs used Stat, which reports a dangling link as
// missing, so staging listed it and discard removed it — deleting a link while
// the receipt said nothing was written.
func TestAStagingAbortKeepsAPreExistingDanglingSymlink(t *testing.T) {
	root := t.TempDir()
	write(t, root, "b.txt", "bee\n")
	if err := os.Symlink("missing", filepath.Join(root, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, err := Apply(root, []Input{
		{Path: "b.txt", Op: "rename", Body: []string{"link/b.txt"}, Lines: -1, Index: 0},
	}, Options{})
	if err == nil {
		t.Fatal("a rename under a dangling symlink was not refused")
	}
	if got, lerr := os.Readlink(filepath.Join(root, "link")); lerr != nil || got != "missing" {
		t.Fatalf("the aborted plan removed or changed a symlink it did not make: %q %v", got, lerr)
	}
	if got := read(t, root, "b.txt"); got != "bee\n" {
		t.Fatalf("b.txt = %q, want it untouched", got)
	}
}

// ADR-066 T1 (Codex review of #207). A leaf the filesystem rejects under a
// parent that does not exist answers "does not exist" at validation; staging
// makes the parent and then asks again, so the plan aborts before the sibling
// edit is written, and the parent it made is taken back.
func TestARenameWhoseLeafIsRejectedUnderANewParentWritesNothing(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", abcde)
	write(t, root, "b.txt", "bee\n")
	res, err := Apply(root, []Input{
		{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"CHANGED"}, Lines: -1, Index: 0},
		{Path: "b.txt", Op: "rename", Body: []string{"new/" + long}, Lines: -1, Index: 1},
	}, Options{})
	if err == nil {
		t.Fatalf("a rejected leaf under a new parent was not reported: %+v", res)
	}
	if got := read(t, root, "a.txt"); got != abcde {
		t.Fatalf("a.txt was written: %q", got)
	}
	if _, serr := os.Stat(filepath.Join(root, "new")); !os.IsNotExist(serr) {
		t.Fatalf("the parent staging made was not taken back: %v", serr)
	}
	if h := hunkFor(t, res, "b.txt"); h.Status != StatusFailed {
		t.Fatalf("rename status %s, want failed", h.Status)
	}
}

// ADR-066 T1. The hunk whose file could not be staged describes no write, so
// it carries no Echo or Balance, like its skipped siblings.
func TestAStagingFailureLeavesNoWriteDetailOnTheFailedHunk(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.go", "x\ny\nz\n")
	real := stageFileFn
	t.Cleanup(func() { stageFileFn = real })
	stageFileFn = func(string, text) (staged, error) { return staged{}, errors.New("staging refused") }
	res, err := Apply(root, []Input{
		{Path: "a.go", Start: 1, End: 1, Op: "replace", Body: []string{"{"}, Lines: -1, Index: 0},
	}, Options{EchoPad: 1})
	if err == nil {
		t.Fatal("a failed stage was not reported")
	}
	h := hunkFor(t, res, "a.go")
	if h.Status != StatusFailed || h.Balance != "" || len(h.Echo) != 0 {
		t.Fatalf("failed hunk = status %s balance %q echo %v, want failed with no write detail", h.Status, h.Balance, h.Echo)
	}
}
