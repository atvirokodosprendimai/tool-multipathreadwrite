package apply

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// ADR-076 T1: a plan path ending in a separator names a directory, and a plan
// edits files — `@@ a.txt/` was cleaned to a.txt and edited it at exit 0. It is
// refused whether or not the path exists.
func TestAHunkPathEndingInASeparatorIsRefused(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "a\n")
	sep := string(filepath.Separator)
	for _, in := range []Input{
		{Path: "a.txt" + sep, Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1},
		{Path: "new" + sep, Op: "create", Body: []string{"x"}, Lines: -1},
	} {
		res, err := Apply(root, []Input{in}, Options{})
		if err != nil {
			t.Fatal(err)
		}
		if res.Applied || res.Failed != 1 || !strings.Contains(res.Hunks[0].Reason, "names a directory") {
			t.Errorf("%s %s was not refused: applied=%v %+v", in.Op, in.Path, res.Applied, res.Hunks)
		}
	}
	if got := read(t, root, "a.txt"); got != "a\n" {
		t.Errorf("a.txt changed: %q", got)
	}
	if exists(t, root, "new") {
		t.Error("a create of new/ made new")
	}
	res, err := Apply(root, []Input{{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1}}, Options{})
	if err != nil || !res.Applied {
		t.Fatalf("a.txt spelled plainly did not apply: %v %+v", err, res.Hunks)
	}
}

// ADR-076 T1: a rename to `d/` made a FILE named d. The destination is refused
// whether d exists or not, and the refusal names the file to write instead.
func TestARenameToADirectorySpellingIsRefused(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "a\n")
	if err := os.Mkdir(filepath.Join(root, "e"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, dest := range []string{"d/", "e/"} {
		res, err := Apply(root, []Input{{Path: "a.txt", Op: "rename", Body: []string{dest}, Lines: -1}}, Options{})
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(filepath.Clean(dest), "a.txt")
		if res.Applied || !strings.Contains(res.Hunks[0].Reason, "names a directory") || !strings.Contains(res.Hunks[0].Reason, want) {
			t.Errorf("rename to %s was not refused naming %s: applied=%v %+v", dest, want, res.Applied, res.Hunks)
		}
	}
	if exists(t, root, "d") || !exists(t, root, "a.txt") {
		t.Fatal("a refused rename moved a.txt")
	}
	res, err := Apply(root, []Input{{Path: "a.txt", Op: "rename", Body: []string{"d/a.txt"}, Lines: -1}}, Options{})
	if err != nil || !res.Applied || read(t, root, "d/a.txt") != "a\n" {
		t.Fatalf("a rename naming the file did not land: %v %+v", err, res.Hunks)
	}
}

// ADR-076 T4: a write through an in-root symlink follows it (ADR-005 §2), and
// the receipt named only the link. It names the file that changed as well; a
// plain write names no target.
func TestAWriteThroughASymlinkNamesItsTarget(t *testing.T) {
	root := t.TempDir()
	write(t, root, "real.txt", "a\n")
	if err := os.Symlink("real.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	res, err := Apply(root, []Input{{Path: "link.txt", Start: 1, End: 1, Op: "replace", Body: []string{"b"}, Lines: -1}}, Options{})
	if err != nil || !res.Applied || len(res.Files) != 1 || res.Files[0].Target != "real.txt" {
		t.Fatalf("write through link.txt: err %v, files %+v", err, res.Files)
	}
	res, err = Apply(root, []Input{{Path: "real.txt", Start: 1, End: 1, Op: "replace", Body: []string{"c"}, Lines: -1}}, Options{})
	if err != nil || !res.Applied || res.Files[0].Target != "" {
		t.Fatalf("a plain write named a target: err %v, files %+v", err, res.Files)
	}
}

// ADR-076 T4: the directories a create or a rename needed were made and not
// named. They are named, parents first; a dry run makes none and names none.
func TestThePlanNamesTheDirectoriesItMade(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "a\n")
	write(t, root, "have/x.txt", "x\n")
	res, err := Apply(root, []Input{
		{Path: "n/deep/c.txt", Op: "create", Body: []string{"c"}, Lines: -1, Index: 0},
		{Path: "a.txt", Op: "rename", Body: []string{"m/a.txt"}, Lines: -1, Index: 1},
		{Path: "have/y.txt", Op: "create", Body: []string{"y"}, Lines: -1, Index: 2},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"m", "n", filepath.Join("n", "deep")}; !res.Applied || !reflect.DeepEqual(res.DirsCreated, want) {
		t.Fatalf("dirs_created = %q, want %q (applied=%v %+v)", res.DirsCreated, want, res.Applied, res.Hunks)
	}
	res, err = Apply(root, []Input{{Path: "z/z.txt", Op: "create", Body: []string{"z"}, Lines: -1}}, Options{DryRun: true})
	if err != nil || len(res.DirsCreated) != 0 || exists(t, root, "z") {
		t.Fatalf("a dry run named or made directories: %v %q", err, res.DirsCreated)
	}
}

// ADR-076 T5: an empty file has no last line whose terminator could be kept,
// so the lines written into it end with a newline, as a created file's do.
// TestTrailingNewlineIsPreserved is the pair: a file that lacks one keeps
// lacking it.
func TestAnInsertIntoAnEmptyFileEndsItsLineWithANewline(t *testing.T) {
	root := t.TempDir()
	write(t, root, "e.txt", "")
	res, err := Apply(root, []Input{{Path: "e.txt", Start: 0, End: 0, Op: "insert-after", Body: []string{"x"}, Lines: -1}}, Options{})
	if err != nil || !res.Applied {
		t.Fatalf("insert into an empty file: %v %+v", err, res.Hunks)
	}
	if got := read(t, root, "e.txt"); got != "x\n" {
		t.Errorf("got %q, want %q", got, "x\n")
	}
}

// ADR-076 T6: a read-only file is a statement by whoever marked it. A replace
// renames a new file over it and an unlink removes the entry, neither of which
// needs the file to be writable, so every op went through at exit 0. Each is
// refused now, bytes and mode unchanged; a writable file is the pair.
func TestAReadOnlyFileIsRefusedForEveryOpThatChangesIt(t *testing.T) {
	for _, in := range []Input{
		{Path: "ro.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1},
		{Path: "ro.txt", Start: 1, End: 1, Op: "insert-after", Body: []string{"X"}, Lines: -1},
		{Path: "ro.txt", Start: 1, End: 1, Op: "delete", Lines: -1},
		{Path: "ro.txt", Op: "unlink", Lines: -1},
		{Path: "ro.txt", Op: "rename", Body: []string{"moved.txt"}, Lines: -1},
	} {
		root := t.TempDir()
		write(t, root, "ro.txt", "a\nb\n")
		p := filepath.Join(root, "ro.txt")
		if err := os.Chmod(p, 0o444); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(p, 0o644) })
		res, err := Apply(root, []Input{in}, Options{})
		if err != nil {
			t.Fatal(err)
		}
		if res.Applied || res.Failed != 1 || !strings.Contains(res.Hunks[0].Reason, "read-only") {
			t.Errorf("%s on a read-only file was not refused: applied=%v %+v", in.Op, res.Applied, res.Hunks)
		}
		if got := read(t, root, "ro.txt"); got != "a\nb\n" || exists(t, root, "moved.txt") {
			t.Errorf("%s changed a read-only file: %q", in.Op, got)
		}
	}
	root := t.TempDir()
	write(t, root, "rw.txt", "a\n")
	res, err := Apply(root, []Input{{Path: "rw.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1}}, Options{})
	if err != nil || !res.Applied {
		t.Fatalf("a writable file was refused: %v %+v", err, res.Hunks)
	}
}

// ADR-076 T6: an unlink moves the directory entry, so it is the entry's own
// mode that counts. A link to a read-only file can be removed — the link goes,
// the file stays read-only and untouched.
func TestAnUnlinkOfALinkToAReadOnlyFileRemovesTheLink(t *testing.T) {
	root := t.TempDir()
	write(t, root, "ro.txt", "a\n")
	p := filepath.Join(root, "ro.txt")
	if err := os.Chmod(p, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(p, 0o644) })
	if err := os.Symlink("ro.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	res, err := Apply(root, []Input{{Path: "link.txt", Op: "unlink", Lines: -1}}, Options{})
	if err != nil || !res.Applied {
		t.Fatalf("unlink of a link to a read-only file was refused: %v %+v", err, res.Hunks)
	}
	if _, err := os.Lstat(filepath.Join(root, "link.txt")); !os.IsNotExist(err) {
		t.Errorf("the link is still there: %v", err)
	}
	if got := read(t, root, "ro.txt"); got != "a\n" {
		t.Errorf("the read-only file changed: %q", got)
	}
}

// Codex review of #237. A commit that failed after an earlier file landed left
// that file's new directories on disk, and dirs_created was set only on success,
// so the receipt of a PARTIALLY APPLIED plan hid them.
func TestAPartialCommitNamesTheDirectoriesItLeft(t *testing.T) {
	root := t.TempDir()
	write(t, root, "b.txt", "b\n")
	failRenames(t, func(_, newpath string) bool { return filepath.Base(newpath) == "b.txt" })
	res, err := Apply(root, []Input{
		{Path: "n/deep/a.txt", Op: "create", Body: []string{"a"}, Lines: -1, Index: 0},
		{Path: "b.txt", Start: 1, End: 1, Op: "replace", Body: []string{"B"}, Lines: -1, Index: 1},
	}, Options{})
	if err == nil || res.Applied {
		t.Fatalf("the injected commit failure did not fail the plan: %v", err)
	}
	if !exists(t, root, "n/deep/a.txt") {
		t.Fatal("the fixture needs a.txt to have landed before b.txt failed")
	}
	if want := []string{"n", filepath.Join("n", "deep")}; !reflect.DeepEqual(res.DirsCreated, want) {
		t.Errorf("dirs_created = %q, want %q: directories on disk left out of the receipt", res.DirsCreated, want)
	}
}

// Codex review of #237. On Windows EvalSymlinks canonicalises case, so a file
// named readme.md in a plan and README.md on disk was reported as reached
// through a link. A file reached by its own name, in any case, has no target.
func TestACaseOnlyDifferenceIsNotALinkTarget(t *testing.T) {
	root := t.TempDir()
	write(t, root, "README.md", "a\n")
	if _, err := os.Stat(filepath.Join(root, "readme.md")); err != nil {
		t.Skip("this filesystem keeps case, so readme.md is another file")
	}
	res, err := Apply(root, []Input{{Path: "readme.md", Start: 1, End: 1, Op: "replace", Body: []string{"b"}, Lines: -1}}, Options{})
	if err != nil || !res.Applied || res.Files[0].Target != "" {
		t.Fatalf("a case-only spelling named a target: %v %+v", err, res.Files)
	}
}

// Review of #237. A line edit judges the file a link reaches, so a replace
// through a link to a read-only file is refused; only an unlink or a rename,
// which move the entry, judge the link itself.
func TestALineEditThroughALinkToAReadOnlyFileIsRefused(t *testing.T) {
	root := t.TempDir()
	write(t, root, "ro.txt", "a\n")
	p := filepath.Join(root, "ro.txt")
	if err := os.Chmod(p, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(p, 0o644) })
	if err := os.Symlink("ro.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	res, err := Apply(root, []Input{{Path: "link.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1}}, Options{})
	if err != nil || res.Applied || !strings.Contains(res.Hunks[0].Reason, "read-only") {
		t.Fatalf("a replace through a link to a read-only file was not refused: %v %+v", err, res.Hunks)
	}
	if got := read(t, root, "ro.txt"); got != "a\n" {
		t.Errorf("ro.txt changed: %q", got)
	}
}

// Review of #237. A create over an existing file is refused because the file
// exists; a read-only mark on it hid that reason behind "chmod u+w".
func TestACreateOverAReadOnlyFileSaysItExists(t *testing.T) {
	root := t.TempDir()
	write(t, root, "ro.txt", "a\n")
	p := filepath.Join(root, "ro.txt")
	if err := os.Chmod(p, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(p, 0o644) })
	res, err := Apply(root, []Input{{Path: "ro.txt", Op: "create", Body: []string{"x"}, Lines: -1}}, Options{})
	if err != nil || res.Applied || strings.Contains(res.Hunks[0].Reason, "read-only") || !strings.Contains(res.Hunks[0].Reason, "exist") {
		t.Fatalf("a create over a read-only file did not say the file exists: %v %+v", err, res.Hunks)
	}
}

// Review of #237. The directory-spelling refusal was carried by the file's
// FIRST hunk, which may be a sibling spelled plainly; the hunk that wrote
// `a.txt/` read as skipped.
func TestTheDirectorySpellingRefusalIsCarriedByTheHunkThatSpelledIt(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "a\nb\n")
	res, err := Apply(root, []Input{
		{Path: "a.txt", Start: 2, End: 2, Op: "replace", Body: []string{"B"}, Lines: -1, Index: 0},
		{Path: "a.txt/", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, Index: 1},
	}, Options{})
	if err != nil || res.Applied {
		t.Fatalf("the plan applied: %v", err)
	}
	if res.Hunks[0].Status != StatusSkipped || res.Hunks[1].Status != StatusFailed || !strings.Contains(res.Hunks[1].Reason, "a.txt/") {
		t.Errorf("want the plain hunk skipped and the a.txt/ hunk failed: %+v", res.Hunks)
	}
}

// Review of #237. A create through an in-root linked directory landed in the
// link's target and named no target: resolving the file whole failed, since it
// did not exist yet. It resolves through the directory that holds it.
func TestACreateThroughALinkedDirectoryNamesItsTarget(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "real"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(root, "linkdir")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	res, err := Apply(root, []Input{{Path: "linkdir/new/x.txt", Op: "create", Body: []string{"x"}, Lines: -1}}, Options{})
	if err != nil || !res.Applied {
		t.Fatalf("the create did not apply: %v %+v", err, res.Hunks)
	}
	if want := filepath.Join("real", "new", "x.txt"); res.Files[0].Target != want {
		t.Errorf("target = %q, want %q", res.Files[0].Target, want)
	}
	if want := []string{filepath.Join("real", "new")}; !reflect.DeepEqual(res.DirsCreated, want) {
		t.Errorf("dirs_created = %q, want %q", res.DirsCreated, want)
	}
}
