package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// The v1.60.0 and v1.61.0 Windows retests: `--create -- 'trail '` was told to put
// `--` before the path, which drops --create. Since ADR-196 the parser hands
// --create its value as typed, so `--create 'trail '` names "trail " itself, and
// `--create -- 'trail '` (the -- taken as --create's value) is refused, writing
// nothing. On Windows ADR-071 refuses a name ending in a space.
func TestTheEdgeWhitespaceHintForCreateKeepsCreate(t *testing.T) {
	root := t.TempDir()
	var code int
	withStdin(t, "x\n", func() { _, code = runIn(t, root, "write", "--no-check", "--create", "--", "trail ") })
	if code != exitUsage {
		t.Errorf("--create -- 'trail ': exit %d, want %d", code, exitUsage)
	}
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Fatalf("a refused --create wrote %v", entries)
	}
	var out string
	withStdin(t, "x\n", func() { out, code = runIn(t, root, "write", "--no-check", "--create", "trail ") })
	if runtime.GOOS == "windows" {
		if code == 0 {
			t.Errorf("--create 'trail ' on Windows exited 0:\n%s", out)
		}
		if _, err := os.Stat(filepath.Join(root, "trail")); err == nil {
			t.Error("--create 'trail ' made trail")
		}
		return
	}
	if b, err := os.ReadFile(filepath.Join(root, "trail ")); code != 0 || err != nil || string(b) != "x\n" {
		t.Errorf("--create 'trail ': exit %d, file %q %v, want 0 and \"trail \" holding x:\n%s", code, b, err, out)
	}
	if _, err := os.Stat(filepath.Join(root, "trail")); err == nil {
		t.Error("--create 'trail ' made trail")
	}
}
