//go:build !windows

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// #727. lineCount counted the lines of a file by name once its digest matched
// the version a read observed: through a link out of the root it opened a file
// outside it. Through the held root it is refused; a link inside the root
// still counts.
func TestLineCountDoesNotOpenAFileOutsideTheRoot(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	const body = "a\nb\n"
	for _, p := range []string{filepath.Join(root, "d", "f.txt"), filepath.Join(outside, "f.txt")} {
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(root, "out")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("d", filepath.Join(root, "in")); err != nil {
		t.Fatal(err)
	}
	sha := shaOfBody727(body)
	for _, p := range []string{"d/f.txt", "in/f.txt"} {
		if n, ok := lineCount(root, filepath.Join(root, filepath.FromSlash(p)), sha); !ok || n != 2 {
			t.Errorf("the pair: %s counted %d, %v", p, n, ok)
		}
	}
	if n, ok := lineCount(root, filepath.Join(root, "out", "f.txt"), sha); ok {
		t.Errorf("a file outside the root was counted: %d", n)
	}
}

func shaOfBody727(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
