//go:build !windows

package read

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #727 (ADR-007). rooted.Resolve decided that d/f.txt is inside the root, and
// read then opened the name: a directory swapped for a link to outside in
// between served bytes from outside --root. The seam fires between the
// decision and the open, so the swap is deterministic.
func TestADirectorySwappedForALinkOutAfterTheBoundaryDecisionServesNothing(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "d", "f.txt"), []byte("INSIDE\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "f.txt"), []byte("OUTSIDE-SECRET\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prev := afterResolveFn
	afterResolveFn = func() {
		afterResolveFn = prev
		if err := os.Rename(filepath.Join(root, "d"), filepath.Join(root, "d.old")); err != nil {
			t.Error(err)
			return
		}
		if err := os.Symlink(outside, filepath.Join(root, "d")); err != nil {
			t.Error(err)
		}
	}
	t.Cleanup(func() { afterResolveFn = prev })
	var out bytes.Buffer
	Run(&out, root, []Spec{{Path: "d/f.txt"}}, Options{})
	if strings.Contains(out.String(), "OUTSIDE-SECRET") {
		t.Fatalf("bytes from outside the root were served:\n%s", out.String())
	}
}

// The pair: a link inside the root that leads to a file inside it still reads
// (ADR-071), relative and absolute.
func TestALinkInsideTheRootStillReads(t *testing.T) {
	root := t.TempDir()
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(real, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(real, "d", "f.txt"), []byte("INSIDE\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("d", filepath.Join(real, "rel")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(real, "d"), filepath.Join(real, "abs")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"rel/f.txt", "abs/f.txt"} {
		var out bytes.Buffer
		Run(&out, root, []Spec{{Path: name}}, Options{})
		if !strings.Contains(out.String(), "INSIDE") {
			t.Errorf("%s did not read through a link inside the root:\n%s", name, out.String())
		}
	}
}
