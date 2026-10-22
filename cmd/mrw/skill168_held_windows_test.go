//go:build windows

package main

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide"
)

// The Codex review of f323c1c and the Windows chaos test of 2026-10-22: a
// SKILL.md another process holds without read sharing is the target's cause,
// exit 1 (ADR-125), as the write engine says of a held target — not exit 2.
func TestSkillDirRefusesAHeldSkillAsTheTargets(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	d := t.TempDir()
	if err := os.MkdirAll(filepath.Join(d, "mrw"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(d, "mrw", "SKILL.md")
	if err := os.WriteFile(p, []byte(guide.Skill("v0.0.1")), 0o644); err != nil {
		t.Fatal(err)
	}
	name, err := syscall.UTF16PtrFromString(p)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(name, syscall.GENERIC_READ, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = syscall.CloseHandle(h) }()
	if _, err := skillRun(t, "skill", "--dir", d); err == nil || exitCode(err) != exitNotApplied {
		t.Errorf("a SKILL.md held without sharing: %v, want exit %d", err, exitNotApplied)
	}
}
