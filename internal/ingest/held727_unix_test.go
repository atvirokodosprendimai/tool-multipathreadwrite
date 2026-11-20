//go:build !windows

package ingest

import (
	"os"
	"path/filepath"
	"testing"
)

// tree727 makes a root holding d/f.txt and another directory holding f.txt, and
// returns a func that swaps d for a link to that directory, once (#727).
func tree727(t *testing.T, inside, secret string) (root string, swap func()) {
	t.Helper()
	real, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(real, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(real, "d", "f.txt"), []byte(inside), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "f.txt"), []byte(secret), 0o644); err != nil {
		t.Fatal(err)
	}
	var done bool
	return real, func() {
		if done {
			return
		}
		done = true
		if err := os.Rename(filepath.Join(real, "d"), filepath.Join(real, "d.old")); err != nil {
			t.Error(err)
			return
		}
		if err := os.Symlink(outside, filepath.Join(real, "d")); err != nil {
			t.Error(err)
		}
	}
}

const doc727 = "*** Begin Patch\n*** Update File: d/f.txt\n@@\n-needle\n+changed\n*** End Patch\n"

// #727. apply_patch locates its old side in the target, resolved and then
// opened by name: a directory swapped for a link out of the root in between
// compiled a plan from a file outside it. The inside file has no such line, so
// a plan that compiles is the leak.
func TestAnApplyPatchTargetUnderADirectorySwappedForALinkOutIsNotRead(t *testing.T) {
	root, swap := tree727(t, "other\n", "needle\n")
	prev := afterTargetResolveFn
	afterTargetResolveFn = func() { afterTargetResolveFn = prev; swap() }
	t.Cleanup(func() { afterTargetResolveFn = prev })
	if out, err := CompileApplyPatch(root, []byte(doc727)); err == nil {
		t.Fatalf("a plan was compiled from a file outside the root:\n%s", out)
	}
}

// The pair: a target behind a link inside the root, relative and absolute,
// compiles.
func TestAnApplyPatchTargetBehindALinkInsideTheRootIsStillRead(t *testing.T) {
	root, _ := tree727(t, "needle\n", "x\n")
	if err := os.Symlink("d", filepath.Join(root, "rel")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "d"), filepath.Join(root, "abs")); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"d", "rel", "abs"} {
		doc := "*** Begin Patch\n*** Update File: " + p + "/f.txt\n@@\n-needle\n+changed\n*** End Patch\n"
		if _, err := CompileApplyPatch(root, []byte(doc)); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
}
