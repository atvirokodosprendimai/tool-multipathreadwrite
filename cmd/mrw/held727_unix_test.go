//go:build !windows

package main

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

// #727. The host-read hook resolved a path and then opened the name: a
// directory swapped for a link out of the root in between licensed lines of a
// file outside it, by their digest.
func TestTheHookDoesNotLicenseAFileUnderADirectorySwappedForALinkOut(t *testing.T) {
	root, swap := tree727(t, "inside\n", "SECRET\n")
	prev := afterHookResolveFn
	afterHookResolveFn = func() { afterHookResolveFn = prev; swap() }
	t.Cleanup(func() { afterHookResolveFn = prev })
	if key, obs, ok := hookLicensed(root, "d/f.txt", "SECRET", 1); ok {
		t.Fatalf("lines of a file outside the root were licensed: %s %+v", key, obs)
	}
}

// The pair: lines of a file behind a link inside the root are licensed.
func TestTheHookStillLicensesAFileBehindALinkInsideTheRoot(t *testing.T) {
	root, _ := tree727(t, "inside\n", "x\n")
	if err := os.Symlink("d", filepath.Join(root, "rel")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "d"), filepath.Join(root, "abs")); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"d/f.txt", "rel/f.txt", "abs/f.txt"} {
		if _, _, ok := hookLicensed(root, p, "inside", 1); !ok {
			t.Errorf("%s: lines shown by the host were not licensed", p)
		}
	}
}
