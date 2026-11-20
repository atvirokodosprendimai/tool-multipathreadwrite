package apply

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func movedTree(t *testing.T) (string, *tree) {
	t.Helper()
	root := t.TempDir()
	tr, err := openTree(root, root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(tr.close)
	return root, tr
}

func hookAfterLink(t *testing.T, f func(from, to string)) {
	t.Helper()
	real := afterLink
	t.Cleanup(func() { afterLink = real })
	afterLink = f
}

// Review of #744. A source another process removes between the link and the
// removal of the old name: the move is done, and the new name is the only one
// the file has, so it is not taken back.
func TestAMoveWhoseSourceVanishedAfterTheLinkKeepsTheFile(t *testing.T) {
	root, tr := movedTree(t)
	write(t, root, "a.txt", "A\n")
	hookAfterLink(t, func(from, _ string) {
		if err := os.Remove(from); err != nil {
			t.Fatal(err)
		}
	})
	if err := tr.renameNew(filepath.Join(root, "a.txt"), filepath.Join(root, "b.txt")); err != nil {
		t.Fatalf("a move whose source went meanwhile failed: %v", err)
	}
	if got := read(t, root, "b.txt"); got != "A\n" {
		t.Errorf("b.txt = %q: the only name the file had was taken back", got)
	}
}

// Review of #744. An editor's atomic save puts a new file at the source between
// the link and the removal: that file is not the one linked, and is not removed.
func TestAMoveLeavesAFileThatReplacedTheSourceAfterTheLink(t *testing.T) {
	root, tr := movedTree(t)
	write(t, root, "a.txt", "OLD\n")
	hookAfterLink(t, func(from, _ string) {
		if err := os.WriteFile(from+".save", []byte("EDITOR\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(from+".save", from); err != nil {
			t.Fatal(err)
		}
	})
	if err := tr.renameNew(filepath.Join(root, "a.txt"), filepath.Join(root, "b.txt")); err != nil {
		t.Fatalf("the move failed: %v", err)
	}
	if got := read(t, root, "a.txt"); got != "EDITOR\n" {
		t.Errorf("a.txt = %q: the editor's file was removed", got)
	}
	if got := read(t, root, "b.txt"); got != "OLD\n" {
		t.Errorf("b.txt = %q, want the file that was moved", got)
	}
}

// Review of #744. Two names for one file on a filesystem that keeps case are a
// hard link, not a respelling: rename(2) onto one reports a success and changes
// nothing, so the move is refused as a taken name.
func TestAMoveOntoAHardLinkOfItselfIsRefused(t *testing.T) {
	root, tr := movedTree(t)
	write(t, root, "a.txt", "A\n")
	if err := os.Link(filepath.Join(root, "a.txt"), filepath.Join(root, "b.txt")); err != nil {
		t.Skipf("no hard links here: %v", err)
	}
	err := tr.renameNew(filepath.Join(root, "a.txt"), filepath.Join(root, "b.txt"))
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("a move onto a hard link of itself: %v, want fs.ErrExist", err)
	}
	if read(t, root, "a.txt") != "A\n" || read(t, root, "b.txt") != "A\n" {
		t.Error("a refused move changed a name")
	}
}
