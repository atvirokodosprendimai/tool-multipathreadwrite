//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The Windows chaos tests of 2026-10-22: --dir 'd1.' named d1 through Win32's
// trailing-dot rule while 'd1 ' was refused by the engine; both are refused
// now, exit 2, and nothing is written.
func TestSkillDirRefusesATrailingDotOnWindows(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	d := t.TempDir()
	if err := os.Mkdir(filepath.Join(d, "d1"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{filepath.Join(d, "d1."), filepath.Join(d, "d1") + " "} {
		if _, err := skillRun(t, "skill", "--dir", dir); err == nil || exitCode(err) != exitUsage {
			t.Errorf("--dir %q: %v, want exit %d", dir, err, exitUsage)
		}
	}
	if _, err := os.Stat(filepath.Join(d, "d1", "mrw")); err == nil {
		t.Error("a skill was written into d1 through a trailing dot or space")
	}
}
