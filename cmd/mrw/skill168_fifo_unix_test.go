//go:build unix

package main

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// ADR-168 Decision 2, the Codex review of 2026-10-22. A SKILL.md that is a FIFO
// — swapped in after --dir's Lstat, or there from the start — is refused through
// regular.Open (ADR-109), exit 1, and never waited on.
func TestSkillDirRefusesAFIFOWithoutWaiting(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	d := t.TempDir()
	if err := os.MkdirAll(filepath.Join(d, "mrw"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(d, "mrw", "SKILL.md"), 0o644); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := skillRun(t, "skill", "--dir", d); done <- err }()
	select {
	case err := <-done:
		if err == nil || exitCode(err) != exitNotApplied {
			t.Errorf("a FIFO SKILL.md: %v, want exit %d", err, exitNotApplied)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("--dir waited on a FIFO SKILL.md")
	}
}
