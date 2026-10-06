package apply

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// names lists dir's entries as the directory spells them.
func names(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range es {
		out = append(out, e.Name())
	}
	return out
}

// failedReason is the reason of the first failed hunk, or "".
func failedReason(res Result) string {
	for _, h := range res.Hunks {
		if h.Status == StatusFailed {
			return h.Reason
		}
	}
	return ""
}

// ADR-129. On a filesystem that folds case, A.txt finds a.txt — the source —
// and the rename was refused "already exists". It applies now, and the
// directory spells the file the way the plan asked.
func TestACaseOnlyRenameApplies(t *testing.T) {
	root := t.TempDir()
	if !caseInsensitiveFS(t, root) {
		t.Skip("this filesystem does not fold case: a.txt and A.txt are two names here")
	}
	write(t, root, "a.txt", "x\n")
	res, err := Apply(root, []Input{{Path: "a.txt", Op: "rename", Body: []string{"A.txt"}, Lines: -1}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Applied {
		t.Fatalf("the case-only rename was refused: %s", failedReason(res))
	}
	if got := names(t, root); !slices.Equal(got, []string{"A.txt"}) {
		t.Fatalf("the directory holds %v, want [A.txt]", got)
	}
	if read(t, root, "A.txt") != "x\n" {
		t.Fatal("the renamed file lost its content")
	}
}

// ADR-129. b.txt hard-linked to a.txt is the same file under a name that
// really exists; POSIX rename of one onto the other does nothing and reports
// success. It stays refused, on every platform.
func TestAHardLinkUnderAnotherNameIsStillRefused(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "x\n")
	if err := os.Link(filepath.Join(root, "a.txt"), filepath.Join(root, "b.txt")); err != nil {
		t.Skipf("hard links unavailable here: %v", err)
	}
	res, err := Apply(root, []Input{{Path: "a.txt", Op: "rename", Body: []string{"b.txt"}, Lines: -1}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied || !strings.Contains(failedReason(res), "already exists") {
		t.Fatalf("a rename onto a hard link was not refused as existing: applied=%v %q", res.Applied, failedReason(res))
	}
	if got := names(t, root); !slices.Equal(got, []string{"a.txt", "b.txt"}) {
		t.Fatalf("the directory holds %v, want both names", got)
	}
}

// ADR-129 Decision 2. d/a.txt → D/a.txt on a folding filesystem names the same
// entry with the same leaf: there is nothing to rename.
func TestARespellingOfOnlyTheDirectoryIsRefused(t *testing.T) {
	root := t.TempDir()
	if !caseInsensitiveFS(t, root) {
		t.Skip("this filesystem does not fold case")
	}
	write(t, root, "d/a.txt", "x\n")
	res, err := Apply(root, []Input{{Path: "d/a.txt", Op: "rename", Body: []string{"D/a.txt"}, Lines: -1}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied || !strings.Contains(failedReason(res), "is the source") {
		t.Fatalf("a directory-only respelling was not refused as the source: applied=%v %q", res.Applied, failedReason(res))
	}
	if got := names(t, filepath.Join(root, "d")); !slices.Equal(got, []string{"a.txt"}) {
		t.Fatalf("d holds %v", got)
	}
}

// ADR-129 Decision 3. A filesystem that reports the rename done and keeps the
// old spelling would make a respelling a silent no-op; the commit checks the
// listing, fails the hunk, and nothing stays written.
func TestARespellingTheFilesystemKeptIsUndone(t *testing.T) {
	root := t.TempDir()
	if !caseInsensitiveFS(t, root) {
		t.Skip("this filesystem does not fold case")
	}
	write(t, root, "a.txt", "x\n")
	prev := commitRenameFn
	commitRenameFn = func(*tree, string, string) error { return nil }
	t.Cleanup(func() { commitRenameFn = prev })
	res, err := Apply(root, []Input{{Path: "a.txt", Op: "rename", Body: []string{"A.txt"}, Lines: -1}}, Options{})
	if err == nil || res.Applied {
		t.Fatalf("a respelling the filesystem kept reported success: err=%v applied=%v", err, res.Applied)
	}
	if !strings.Contains(failedReason(res), "kept the old spelling") {
		t.Fatalf("the failed hunk does not say the spelling was kept: %q", failedReason(res))
	}
	for _, f := range res.Files {
		if f.Written {
			t.Fatalf("%s reported written after the undo", f.Path)
		}
	}
	if got := names(t, root); !slices.Equal(got, []string{"a.txt"}) {
		t.Fatalf("the directory holds %v", got)
	}
}

// The Codex review of ADR-129, finding 2. B.txt hard-linked to a.txt, and a
// plan asking a.txt → b.txt: on a folding filesystem b.txt finds B.txt, the
// same file, and no entry is spelled b.txt — yet renaming onto it does nothing.
// Another link to the source in the directory refuses the respelling.
func TestAHardLinkUnderAnotherSpellingIsRefused(t *testing.T) {
	root := t.TempDir()
	if !caseInsensitiveFS(t, root) {
		t.Skip("this filesystem does not fold case")
	}
	write(t, root, "a.txt", "x\n")
	if err := os.Link(filepath.Join(root, "a.txt"), filepath.Join(root, "B.txt")); err != nil {
		t.Skipf("hard links unavailable here: %v", err)
	}
	res, err := Apply(root, []Input{{Path: "a.txt", Op: "rename", Body: []string{"b.txt"}, Lines: -1}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied || !strings.Contains(failedReason(res), "already exists") {
		t.Fatalf("a rename onto another spelling of a hard link was not refused: applied=%v %q", res.Applied, failedReason(res))
	}
	if got := names(t, root); !slices.Equal(got, []string{"B.txt", "a.txt"}) {
		t.Fatalf("the directory holds %v", got)
	}
}

// The Codex review of ADR-129, finding 1. Foo.txt on disk, a plan naming it
// foo.txt: an undo would put the file back as foo.txt, a name it never had. The
// source must be named as the directory lists it.
func TestASourceNamedInAnotherSpellingIsRefused(t *testing.T) {
	root := t.TempDir()
	if !caseInsensitiveFS(t, root) {
		t.Skip("this filesystem does not fold case")
	}
	write(t, root, "Foo.txt", "x\n")
	res, err := Apply(root, []Input{{Path: "foo.txt", Op: "rename", Body: []string{"FOO.txt"}, Lines: -1}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied || !strings.Contains(failedReason(res), "not spelled that way") {
		t.Fatalf("a source named in another spelling was not refused: applied=%v %q", res.Applied, failedReason(res))
	}
	if got := names(t, root); !slices.Equal(got, []string{"Foo.txt"}) {
		t.Fatalf("the directory holds %v", got)
	}
}

// The Codex review of ADR-129, finding 3. Content commits before path ops, so
// a respelling found kept after a content edit landed leaves that edit
// written: the receipt says so (PARTIALLY APPLIED), and the respelling itself
// is undone.
func TestAKeptRespellingAfterAContentEditIsReportedPartial(t *testing.T) {
	root := t.TempDir()
	if !caseInsensitiveFS(t, root) {
		t.Skip("this filesystem does not fold case")
	}
	write(t, root, "a.txt", "x\n")
	write(t, root, "c.txt", "c\n")
	prev := commitRenameFn
	commitRenameFn = func(tr *tree, from, to string) error {
		if filepath.Base(to) == "A.txt" {
			return nil // a filesystem that keeps the old spelling
		}
		return prev(tr, from, to)
	}
	t.Cleanup(func() { commitRenameFn = prev })
	res, err := Apply(root, []Input{
		{Path: "c.txt", Start: 1, End: 1, Op: "replace", Body: []string{"C"}, Lines: -1, Index: 0},
		{Path: "a.txt", Op: "rename", Body: []string{"A.txt"}, Lines: -1, Index: 1},
	}, Options{Force: true})
	if err == nil || res.Applied {
		t.Fatalf("a kept respelling reported success: err=%v applied=%v", err, res.Applied)
	}
	wrote := false
	for _, f := range res.Files {
		if f.Path == "c.txt" && f.Written {
			wrote = true
		}
	}
	if !wrote || read(t, root, "c.txt") != "C\n" {
		t.Fatalf("the landed content edit is not reported written: files=%+v", res.Files)
	}
	if got := names(t, root); !slices.Equal(got, []string{"a.txt", "c.txt"}) {
		t.Fatalf("the directory holds %v", got)
	}
}
