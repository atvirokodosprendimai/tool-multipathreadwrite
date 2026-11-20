//go:build !windows

package read

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// swapForLink727 renames dir d under root and puts a link to outside in its
// place, once, the first time the returned func is called.
func swapForLink727(t *testing.T, root, outside string) func() {
	t.Helper()
	var done bool
	return func() {
		if done {
			return
		}
		done = true
		if err := os.Rename(filepath.Join(root, "d"), filepath.Join(root, "d.old")); err != nil {
			t.Error(err)
			return
		}
		if err := os.Symlink(outside, filepath.Join(root, "d")); err != nil {
			t.Error(err)
		}
	}
}

func tree727(t *testing.T, inside, secret string) (root, outside string) {
	t.Helper()
	root, outside = t.TempDir(), t.TempDir()
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	root = real
	if err := os.Mkdir(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "d", "f.txt"), []byte(inside), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "f.txt"), []byte(secret), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, outside
}

// #727 (ADR-007). The --grep walk reads a discovered file by name after the
// walk found it: a directory swapped for a link out of the root in between made
// the match come from bytes outside it. The inside file has no match; the
// outside one does, so a spec for d/f.txt is the leak.
func TestAWalkedFileUnderADirectorySwappedForALinkOutIsNotMatched(t *testing.T) {
	// whole: the read is asked for with no budget, the path a file over the
	// scan budget takes (readCappedIn); otherwise it is the budgeted one.
	for _, whole := range []bool{false, true} {
		root, outside := tree727(t, "nothing here\n", "needle\n")
		swap := swapForLink727(t, root, outside)
		prev := scanRead
		scanRead = func(r, path string, budget int64) ([]byte, error) {
			swap()
			if whole {
				budget = 0
			}
			return prev(r, path, budget)
		}
		specs, _, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("needle")})
		scanRead = prev
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range specs {
			t.Errorf("whole %v: a match served from outside the root: %+v", whole, s)
		}
	}
}

// The pair: a file named through a link inside the root is still matched.
func TestAWalkedNamedFileThroughALinkInsideTheRootIsStillMatched(t *testing.T) {
	root, _ := tree727(t, "needle\n", "x\n")
	if err := os.Symlink("d", filepath.Join(root, "in")); err != nil {
		t.Fatal(err)
	}
	specs, _, err := Walk(root, []string{"in/f.txt"}, WalkOptions{Pattern: regexp.MustCompile("needle")})
	if err != nil || len(specs) != 1 {
		t.Fatalf("specs %+v err %v, want the one match", specs, err)
	}
}

// #727. The ast-grep CR-only probe reads a hit's file whole, after the boundary
// let it through: swapped for a link out, it classified the outside file. The
// outside file is CR-only, so a problem naming "\r alone" is the leak.
func TestTheAstGrepProbeDoesNotReadThroughADirectorySwappedForALinkOut(t *testing.T) {
	root, outside := tree727(t, "package a\n", "x\ry\rz")
	installRecordingAstGrep(t, hit096("d/f.txt"))
	swap := swapForLink727(t, root, outside)
	prev := afterProbeResolveFn
	afterProbeResolveFn = swap
	t.Cleanup(func() { afterProbeResolveFn = prev })
	_, problems, err := AstGrep(root, nil, "a", nil, AstGrepOptions{NoIgnore: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range problems {
		if p.Path == "d/f.txt" {
			t.Errorf("the probe classified a file outside the root: %+v", p)
		}
	}
}

// The pair: the probe still reads a CR-only file inside the root, through a link
// inside it.
func TestTheAstGrepProbeStillReadsThroughALinkInsideTheRoot(t *testing.T) {
	root, _ := tree727(t, "x\ry\rz", "x\n")
	if err := os.Symlink("d", filepath.Join(root, "in")); err != nil {
		t.Fatal(err)
	}
	installRecordingAstGrep(t, hit096("in/f.txt"))
	_, problems, err := AstGrep(root, nil, "a", nil, AstGrepOptions{NoIgnore: true})
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, p := range problems {
		found = found || strings.Contains(p.Reason, "\\r alone")
	}
	if !found {
		t.Errorf("the CR-only file behind a link inside the root was not probed: %+v", problems)
	}
}
