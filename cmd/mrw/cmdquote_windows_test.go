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

// ADR-078 T1. A padded-path refusal suggested `mrw read -- ' x'`, and cmd.exe
// keeps single quotes as characters, so pasting it there named a path with
// quotes in it. On Windows the refusal also names the cmd.exe form, and that
// form, run through cmd.exe, serves the file.
func TestThePaddedPathFixNamesTheCmdExeForm(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, " x"), []byte("padded\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runIn(t, root, "read", " x")
	want := `(in cmd.exe: mrw read -- " x")`
	if code != exitUsage || !strings.Contains(out, want) {
		t.Fatalf("exit %d, want the cmd.exe form %s:\n%s", code, want, out)
	}
	// The name's own apostrophe survives: rewriting every ' in the POSIX form
	// turned it into a quote (the review of #239).
	if err := os.WriteFile(filepath.Join(root, " O'Brien"), []byte("apostrophe\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, code := runIn(t, root, "read", " O'Brien"); code != exitUsage || !strings.Contains(out, `(in cmd.exe: mrw read -- " O'Brien")`) {
		t.Fatalf("exit %d, the cmd.exe form lost the apostrophe:\n%s", code, out)
	}
	exe := mrwExe(t)
	c := exec.Command("cmd")
	c.SysProcAttr = &syscall.SysProcAttr{CmdLine: `cmd /s /c ""` + exe + `" -C "` + root + `" read -- " x""`}
	c.Env = append(os.Environ(), "XDG_STATE_HOME="+t.TempDir())
	b, err := c.CombinedOutput()
	if err != nil || !strings.Contains(string(b), "| padded") {
		t.Errorf("the cmd.exe form did not serve the file: %v\n%s", err, b)
	}
}
