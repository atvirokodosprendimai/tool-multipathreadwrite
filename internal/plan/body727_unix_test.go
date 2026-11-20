//go:build !windows

package plan

import (
	"os"
	"path/filepath"
	"testing"
)

// #727 (ADR-007). body=@ resolved its file by name and then opened it by name:
// a directory swapped for a link out of the root in between put bytes from
// outside the root into the plan's body. The pair: the same file unswapped, and
// one behind a link inside the root, load.
func TestABodyFileUnderADirectorySwappedForALinkOutIsNotLoaded(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(real, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(real, "d", "b.txt"), []byte("inside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "b.txt"), []byte("OUTSIDE-SECRET\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("d", filepath.Join(real, "in")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(real, "d"), filepath.Join(real, "abs")); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"d/b.txt", "in/b.txt", "abs/b.txt"} {
		hs := []Hunk{{BodyFile: p, SrcLine: 1}}
		if err := LoadBodyFiles(root, hs); err != nil || len(hs[0].Body) != 1 || hs[0].Body[0] != "inside" {
			t.Fatalf("the pair: %s did not load: %v %q", p, err, hs[0].Body)
		}
	}
	prev := afterBodyResolveFn
	afterBodyResolveFn = func() {
		afterBodyResolveFn = prev
		if err := os.Rename(filepath.Join(real, "d"), filepath.Join(real, "d.old")); err != nil {
			t.Error(err)
			return
		}
		if err := os.Symlink(outside, filepath.Join(real, "d")); err != nil {
			t.Error(err)
		}
	}
	t.Cleanup(func() { afterBodyResolveFn = prev })
	hs := []Hunk{{BodyFile: "d/b.txt", SrcLine: 1}}
	if err := LoadBodyFiles(root, hs); err == nil && len(hs[0].Body) > 0 {
		t.Fatalf("a body file outside the root was loaded: %q", hs[0].Body)
	}
}
