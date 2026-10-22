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

// The Codex review of f323c1c: a FIFO there before the run is refused by the
// Lstat check and never reaches the read, so the read is driven on its own —
// the case of a FIFO swapped in after the Lstat. readSkillFile must refuse it,
// exit 1, without waiting for a writer.
func TestReadSkillFileRefusesAFIFOWithoutWaiting(t *testing.T) {
	p := filepath.Join(t.TempDir(), "SKILL.md")
	if err := syscall.Mkfifo(p, 0o644); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := readSkillFile(p); done <- err }()
	select {
	case err := <-done:
		if err == nil || exitCode(err) != exitNotApplied {
			t.Errorf("readSkillFile on a FIFO: %v, want exit %d", err, exitNotApplied)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("readSkillFile waited on a FIFO")
	}
}

// The Codex review of 8c1f08b: after a refused write, --dir re-reads the file to
// tell a race from a refusal. A FIFO there is no race, and the re-read must not
// wait on it.
func TestChangedSinceDoesNotWaitOnAFIFO(t *testing.T) {
	p := filepath.Join(t.TempDir(), "SKILL.md")
	if err := syscall.Mkfifo(p, 0o644); err != nil {
		t.Fatal(err)
	}
	done := make(chan bool, 1)
	go func() { done <- changedSince(p, nil) }()
	select {
	case raced := <-done:
		if raced {
			t.Error("a FIFO was taken for a racing --dir")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("changedSince waited on a FIFO")
	}
}
