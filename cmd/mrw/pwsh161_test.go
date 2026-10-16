package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ADR-161 Decision 13. PowerShell appends CRLF to what it pipes, so a plan
// whose lines end in LF arrived with a last line holding only CR, and the
// write added a blank line (found on Windows by a peer session probing T8).
// ADR-142's rule drops it from a plan on standard input and from `read -`, and
// stderr says so; an all-CRLF plan is applied as it always was, with no note.
func TestAPipedCRLFIsDroppedFromStandardInput(t *testing.T) {
	root, run := subRun(t)
	subTree(t, root, map[string]string{"f.txt": "a\nb\n"})
	if _, e, c := run("", "read", "f.txt"); c != 0 {
		t.Fatalf("read f.txt: exit %d\n%s", c, e)
	}
	const note = "PowerShell appends"
	out, errOut, code := run("@@ f.txt 1 replace\nA\n\r\n", "write", "--no-check", "-")
	if code != 0 || subFile(t, root, "f.txt") != "A\nb\n" || !strings.Contains(errOut, note) {
		t.Errorf("a plan with PowerShell's CRLF: exit %d, f.txt %q; want 0, \"A\\nb\\n\" and the note:\n%s%s", code, subFile(t, root, "f.txt"), out, errOut)
	}

	// The pair: an all-CRLF plan is unchanged by this rule, and gets no note.
	out, errOut, code = run("@@ f.txt 2 replace\r\nB\r\n", "write", "--no-check", "-")
	if code != 0 || subFile(t, root, "f.txt") != "A\nB\n" || strings.Contains(errOut, note) {
		t.Errorf("an all-CRLF plan: exit %d, f.txt %q; want 0, \"A\\nB\\n\" and no note:\n%s%s", code, subFile(t, root, "f.txt"), out, errOut)
	}

	// read - drops it too: two lines, not three.
	out, errOut, code = run("x\ny\n\r\n", "read", "-")
	if code != 0 || !strings.Contains(out, "==> -  2L") || !strings.Contains(errOut, note) {
		t.Errorf("read - of PowerShell's output: exit %d, want 0, two lines and the note:\n%s%s", code, out, errOut)
	}
}

// The trim streams: an unknown --format is refused at once though standard input
// stays open (it used to wait for EOF once the plan was read whole first), and a
// plan FILE named `<stdin>` is that file, not standard input (the Codex review
// of T10).
func TestAPipedPlanStreamsAndAFileNamedStdinIsAFile(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "mrw.exe")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	root, state := t.TempDir(), t.TempDir()
	cmd := exec.Command(bin, "--root", root, "write", "--format=bogus", "--json", "-")
	cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+state)
	// An *os.File, so the child holds the pipe itself: with an io.Reader, Wait
	// would wait for exec's copy of it, which only ends at EOF.
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pw.Close() }()
	cmd.Stdin = pr
	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	_ = pr.Close() // the child has its copy; the write end stays open
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 2 {
			t.Errorf("--format=bogus: %v, want exit 2", err)
		}
	case <-time.After(20 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("--format=bogus waited on an open standard input")
	}

	if runtime.GOOS == "windows" { // < and > cannot be in a Windows file name
		return
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "g.txt"), []byte("g\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "<stdin>"), []byte("@@ g.txt 1 replace\nG\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	read := exec.Command(bin, "--root", root, "read", "g.txt")
	read.Env = append(os.Environ(), "XDG_STATE_HOME="+state)
	if out, err := read.CombinedOutput(); err != nil {
		t.Fatalf("read g.txt: %v\n%s", err, out)
	}
	w := exec.Command(bin, "--root", root, "write", "--no-check", "<stdin>")
	w.Dir, w.Env = dir, append(os.Environ(), "XDG_STATE_HOME="+state)
	w.Stdin = strings.NewReader("@@ g.txt 1 replace\nFROM STDIN\n")
	if out, err := w.CombinedOutput(); err != nil {
		t.Fatalf("write <stdin> (a file): %v\n%s", err, out)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "g.txt")); string(b) != "G\n" {
		t.Errorf("a plan file named <stdin> was not the plan applied: g.txt = %q", b)
	}
}
