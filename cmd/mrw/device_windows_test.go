//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-076 T2 end to end, and ADR-081. NUL is the NUL device on every Windows,
// so a read of it was served as an empty file and a plan could write to it at
// exit 0. Every reserved name is refused by name, with or without an
// extension, whatever this build's GetFullPathName says: v1.27.0 asked it, and
// on Windows 11 (26200) it created con, nul.txt and COM1.txt as files its own
// unlink and PowerShell 5 could not reach (a Windows peer, 2026-09-26).
func TestADeviceNameIsRefusedNotReadAsAnEmptyFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"NUL", "con", "nul.txt", "COM1.txt", "aux.go", "CONOUT$"} {
		if out, code := runIn(t, root, "read", name); code == 0 || !strings.Contains(out, "device") || strings.Contains(out, "0L") {
			t.Errorf("read %s: exit %d:\n%s", name, code, out)
		}
	}
	plans := t.TempDir()
	for name, plan := range map[string]string{
		"create-NUL":      "@@ NUL 0 create\nx\n",
		"create-con":      "@@ con 0 create\nx\n",
		"create-nul.txt":  "@@ nul.txt 0 create\nx\n",
		"create-COM1.txt": "@@ COM1.txt 0 create\nx\n",
		"rename-NUL":      "@@ a.txt - rename\nNUL\n",
		"rename-lpt1.log": "@@ a.txt - rename\nlpt1.log\n",
	} {
		p := filepath.Join(plans, name+".mrw")
		if err := os.WriteFile(p, []byte(plan), 0o644); err != nil {
			t.Fatal(err)
		}
		out, code := runIn(t, root, "write", "--no-check", "--force", p)
		if code != exitNotApplied || !strings.Contains(out, "device") {
			t.Errorf("%s: exit %d, want %d naming the device:\n%s", name, code, exitNotApplied, out)
		}
	}
	if b, err := os.ReadFile(filepath.Join(root, "a.txt")); err != nil || string(b) != "a\n" {
		t.Errorf("a.txt changed or went: %q %v", b, err)
	}
	if entries, _ := os.ReadDir(root); len(entries) != 1 {
		t.Errorf("a refused plan left files behind: %v", entries)
	}
	// The pair: a name that only starts like one is a file.
	p := filepath.Join(plans, "file.mrw")
	if err := os.WriteFile(p, []byte("@@ console.txt 0 create\nx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, code := runIn(t, root, "write", "--no-check", p); code != 0 {
		t.Errorf("console.txt, not a device name, was refused: exit %d:\n%s", code, out)
	}
}

// Codex review of #237. The device check took its candidate from the path as
// written, so `NUL/.` and `NUL/x/..` — which name NUL once cleaned — passed it.
func TestADeviceNameBehindADotComponentIsRefused(t *testing.T) {
	root := t.TempDir()
	for _, spec := range []string{"NUL/.", `NUL\x\..`} {
		if out, code := runIn(t, root, "read", spec); code == 0 || !strings.Contains(out, "device") {
			t.Errorf("read %s: exit %d, want the device refusal:\n%s", spec, code, out)
		}
	}
}
