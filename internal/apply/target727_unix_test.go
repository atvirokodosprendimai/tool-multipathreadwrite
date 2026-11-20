//go:build !windows

package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #727 (ADR-007). Apply resolved a hunk's file by name and then loaded it by
// name: a directory swapped for a link out of the root in between loaded bytes
// from outside it, which end up in the receipt (the hunk's before lines and
// preview). The load goes through a root held open, so the swap is refused.
func TestATargetUnderADirectorySwappedForALinkOutIsNotLoaded(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, root, "d/f.txt", "inside\n")
	write(t, outside, "f.txt", "OUTSIDE-SECRET\n")
	prev := afterTargetResolveFn
	afterTargetResolveFn = func() {
		afterTargetResolveFn = prev
		if err := os.Rename(filepath.Join(root, "d"), filepath.Join(root, "d.old")); err != nil {
			t.Error(err)
			return
		}
		if err := os.Symlink(outside, filepath.Join(root, "d")); err != nil {
			t.Error(err)
		}
	}
	t.Cleanup(func() { afterTargetResolveFn = prev })
	res, err := Apply(root, []Input{{Path: "d/f.txt", Op: "replace", Start: 1, End: 1, Anchor: "zzz", Body: []string{"x"}, Lines: -1}}, Options{Force: true})
	if res.Applied {
		t.Fatalf("a plan for a file swapped for a link out applied: %+v", res.Hunks)
	}
	if got := fmt.Sprintf("%+v %v", res, err); strings.Contains(got, "OUTSIDE-SECRET") {
		t.Errorf("bytes from outside the root reached the receipt: %s", got)
	}
	if b, _ := os.ReadFile(filepath.Join(outside, "f.txt")); string(b) != "OUTSIDE-SECRET\n" {
		t.Errorf("the file outside the root was written: %q", b)
	}
}

// The pair: a file behind a link inside the root loads and is edited.
func TestATargetThroughALinkInsideTheRootIsStillLoaded(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, root, "d/f.txt", "inside\n")
	if err := os.Symlink("d", filepath.Join(root, "in")); err != nil {
		t.Fatal(err)
	}
	res, err := Apply(root, []Input{{Path: "in/f.txt", Op: "replace", Start: 1, End: 1, Body: []string{"x"}, Lines: -1}}, Options{Force: true})
	if err != nil || !res.Applied {
		t.Fatalf("an edit through a link inside the root: %v %+v", err, res.Hunks)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "d", "f.txt")); string(b) != "x\n" {
		t.Errorf("d/f.txt is %q, want the edit", b)
	}
}
