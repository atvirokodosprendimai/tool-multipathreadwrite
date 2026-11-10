//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// ADR-078 T1, retargeted by ADR-196. A padded-path refusal used to name a
// cmd.exe form, since cmd.exe keeps single quotes as characters. The refusal is
// gone: a name with a leading space reaches the read as typed, from Go and from
// cmd.exe alike, apostrophe included.
func TestThePaddedPathFixNamesTheCmdExeForm(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, " x"), []byte("padded\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, code := runIn(t, root, "read", " x"); code != 0 || !strings.Contains(out, "| padded") {
		t.Fatalf("exit %d, want 0 serving \" x\":\n%s", code, out)
	}
	if err := os.WriteFile(filepath.Join(root, " O'Brien"), []byte("apostrophe\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, code := runIn(t, root, "read", " O'Brien"); code != 0 || !strings.Contains(out, "| apostrophe") {
		t.Fatalf("exit %d, want 0 serving \" O'Brien\":\n%s", code, out)
	}
	exe := mrwExe(t)
	c := exec.Command("cmd")
	c.SysProcAttr = &syscall.SysProcAttr{CmdLine: `cmd /s /c ""` + exe + `" -C "` + root + `" read " x""`}
	c.Env = append(os.Environ(), "XDG_STATE_HOME="+t.TempDir())
	b, err := c.CombinedOutput()
	if err != nil || !strings.Contains(string(b), "| padded") {
		t.Errorf("cmd.exe did not serve \" x\": %v\n%s", err, b)
	}
}
