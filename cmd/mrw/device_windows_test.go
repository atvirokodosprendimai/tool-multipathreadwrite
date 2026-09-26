//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// ADR-076 T2 end to end. NUL is the NUL device on every Windows, so it needs no
// setup: a read of it was served as an empty file, and a plan could write to it
// at exit 0. A name that only looks like a device is asked of the OS, because
// Windows 11 opens nul.bin as a file — and then it is written like one.
func TestADeviceNameIsRefusedNotReadAsAnEmptyFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, code := runIn(t, root, "read", "NUL"); code == 0 || !strings.Contains(out, "device") || strings.Contains(out, "0L") {
		t.Errorf("read NUL: exit %d:\n%s", code, out)
	}
	plans := t.TempDir()
	for name, plan := range map[string]string{
		"create": "@@ NUL 0 create\nx\n",
		"rename": "@@ a.txt - rename\nCON\n",
	} {
		p := filepath.Join(plans, name+".mrw")
		if err := os.WriteFile(p, []byte(plan), 0o644); err != nil {
			t.Fatal(err)
		}
		out, code := runIn(t, root, "write", "--no-check", "--force", p)
		if code != exitNotApplied || !strings.Contains(out, "device") {
			t.Errorf("%s naming a device: exit %d, want %d naming the device:\n%s", name, code, exitNotApplied, out)
		}
	}
	if b, err := os.ReadFile(filepath.Join(root, "a.txt")); err != nil || string(b) != "a\n" {
		t.Errorf("a.txt changed or went: %q %v", b, err)
	}
	full, err := syscall.FullPath(filepath.Join(root, "nul.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(full, `\\.\`) {
		t.Skipf("this Windows opens nul.bin as %s, a device; the file half has nothing to show", full)
	}
	p := filepath.Join(plans, "file.mrw")
	if err := os.WriteFile(p, []byte("@@ nul.bin 0 create\nx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, code := runIn(t, root, "write", "--no-check", p); code != 0 {
		t.Errorf("nul.bin, a file on this Windows, was refused: exit %d:\n%s", code, out)
	}
}
