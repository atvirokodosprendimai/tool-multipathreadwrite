//go:build !windows

package rooted

import (
	"os"
	"path/filepath"
	"testing"
)

// #727. OpenRead opens beneath a held root, so a directory swapped for a link
// out of the root AFTER the last name-based step, in the gap openGapFn marks,
// is refused by the open itself and no descriptor is returned. The pair: the
// same file with no swap opens, and so does one behind a link inside the root.
func TestOpenReadRefusesADirectorySwappedForALinkInTheGap(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(real, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{filepath.Join(real, "d", "f.txt"), filepath.Join(outside, "f.txt")} {
		if err := os.WriteFile(p, []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("d", filepath.Join(real, "in")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"d/f.txt", "in/f.txt"} {
		f, _, err := OpenRead(root, filepath.Join(real, filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("%s did not open: %v", name, err)
		}
		_ = f.Close()
	}
	prev := openGapFn
	openGapFn = func() {
		openGapFn = prev
		if err := os.Rename(filepath.Join(real, "d"), filepath.Join(real, "d.old")); err != nil {
			t.Error(err)
			return
		}
		if err := os.Symlink(outside, filepath.Join(real, "d")); err != nil {
			t.Error(err)
		}
	}
	t.Cleanup(func() { openGapFn = prev })
	if f, _, err := OpenRead(root, filepath.Join(real, "d", "f.txt")); err == nil {
		_ = f.Close()
		t.Fatal("a file outside the root was opened through a directory swapped for a link")
	}
}
