//go:build !windows

package apply

import (
	"os"
	"path/filepath"
	"testing"
)

// #727 (ADR-007). loadSource resolved a from= source by name and then opened it
// by name: a directory swapped for a link out of the root in between loaded
// bytes from outside the root, which the plan then judged by digest and could
// copy from. The source is opened through a held root, so the swap is refused.
func TestAFromSourceUnderADirectorySwappedForALinkOutIsNotLoaded(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, root, "d/a.txt", "inside\n")
	write(t, outside, "a.txt", "OUTSIDE-SECRET\n")
	if f := loadSource(root, "d/a.txt"); f.why != "" || len(f.text.lines) != 1 || f.text.lines[0] != "inside" {
		t.Fatalf("the pair: a source inside the root did not load: %+v", f)
	}
	prev := afterSourceResolveFn
	afterSourceResolveFn = func() {
		afterSourceResolveFn = prev
		if err := os.Rename(filepath.Join(root, "d"), filepath.Join(root, "d.old")); err != nil {
			t.Error(err)
			return
		}
		if err := os.Symlink(outside, filepath.Join(root, "d")); err != nil {
			t.Error(err)
		}
	}
	t.Cleanup(func() { afterSourceResolveFn = prev })
	f := loadSource(root, "d/a.txt")
	if len(f.text.lines) != 0 || f.sha != "" || f.why == "" {
		t.Fatalf("a source outside the root was loaded: %+v", f)
	}
}
