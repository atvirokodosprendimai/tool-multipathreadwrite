//go:build !windows

package mcp

import (
	"crypto/sha256"
	"encoding/hex"
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

// #727. countFileLines, staleEnd and currentSHA resolved a path and then opened
// the name: a directory swapped for a link out of the root in between made them
// count, hash and size a file outside it. They share the seam.
func TestMcpContentReadsRefuseADirectorySwappedForALinkOut(t *testing.T) {
	const secret = "s1\ns2\ns3\ns4\ns5\n"
	for name, call := range map[string]func(root string) (string, error){
		"countFileLines": func(root string) (string, error) {
			n, err := countFileLines(root, "d/f.txt")
			return string(rune('0' + n)), err
		},
		"currentSHA": func(root string) (string, error) { return currentSHA(root, "d/f.txt") },
		"staleEnd": func(root string) (string, error) {
			return string(rune('0' + staleEnd(root, "d/f.txt", "1-$", 1))), nil
		},
	} {
		root, swap := tree727(t, "one\n", secret)
		prev := afterResolveFn
		afterResolveFn = func() { afterResolveFn = prev; swap() }
		got, err := call(root)
		afterResolveFn = prev
		leaked := map[string]string{"countFileLines": "5", "staleEnd": "5"}[name]
		if name == "currentSHA" {
			if err == nil && got == shaOf727(secret) {
				t.Errorf("%s hashed a file outside the root", name)
			}
			continue
		}
		if err == nil && got == leaked && got != "" {
			t.Errorf("%s counted the lines of a file outside the root: %s", name, got)
		}
	}
}

// The pair: the same reads through a link inside the root, relative and
// absolute, still see the file.
func TestMcpContentReadsStillFollowALinkInsideTheRoot(t *testing.T) {
	root, _ := tree727(t, "one\ntwo\n", "x\n")
	if err := os.Symlink("d", filepath.Join(root, "rel")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "d"), filepath.Join(root, "abs")); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"d/f.txt", "rel/f.txt", "abs/f.txt"} {
		if n, err := countFileLines(root, p); err != nil || n != 2 {
			t.Errorf("countFileLines(%s) = %d, %v, want 2", p, n, err)
		}
		if sha, err := currentSHA(root, p); err != nil || sha != shaOf727("one\ntwo\n") {
			t.Errorf("currentSHA(%s) = %s, %v", p, sha, err)
		}
	}
}

func shaOf727(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
