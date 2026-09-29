//go:build unix

package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// ADR-095 T2. A timeout reaches the check of a nested mrw. The outer check
// runs a built mrw on an inner tree, `sh` kept the leader by the trailing
// `true`, and the inner check sleeps; the outer check times out. SIGKILL to
// the outer group killed the nested mrw before it could stop its own check,
// whose sleep ran on; TERM first lets it stop that check before it goes.
func TestATimedOutCheckStopsTheCheckOfTheMrwItRan(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	bin := filepath.Join(t.TempDir(), "mrw")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	root := grepTree(t, map[string]string{
		".quality-harness.json":       "{\"check\":\"'" + bin + "' -C inner check --full; true\",\"timeout_seconds\":3}",
		"inner/.quality-harness.json": "{\"check\":\"echo $$ > pid; exec sleep 300\"}",
	})
	pidFile := filepath.Join(root, "inner", "pid")
	var pid int
	t.Cleanup(func() {
		if pid > 0 {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	})
	out, code := runIn(t, root, "check", "--full")
	if code != exitCheckFailed || !strings.Contains(out, "timed out") {
		t.Fatalf("exit %d, want %d and timed out:\n%s", code, exitCheckFailed, out)
	}
	b, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("the inner check never started: %v\n%s", err, out)
	}
	if pid, err = strconv.Atoi(strings.TrimSpace(string(b))); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for syscall.Kill(pid, 0) == nil {
		if time.Now().After(deadline) {
			t.Fatalf("the inner check's sleep %d outlived the outer timeout", pid)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// ADR-095 T2. An ast-grep that answers a TERM with a valid hit and exit 0 was
// stopped at its bound, not answered: it is reported timed out, exit 2, and
// the hit is not served.
func TestAnAstGrepThatAnswersOnTermStillTimesOut(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\nfunc A() {}\n"})
	dir := t.TempDir()
	fake := "#!/bin/sh\ntrap 'printf \"%s\" \"[{\\\"file\\\":\\\"a.go\\\",\\\"range\\\":{\\\"start\\\":{\\\"line\\\":1},\\\"end\\\":{\\\"line\\\":1}}}]\"; exit 0' TERM\nsleep 30 &\nwait\n"
	if err := os.WriteFile(filepath.Join(dir, "ast-grep"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if os.Getenv("XDG_STATE_HOME") == "" {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
	}
	var sink bytes.Buffer
	cmd := rootCommand()
	cmd.Writer, cmd.ErrWriter = &sink, &sink
	var err error
	out, _ := captureStdout(t, func() error {
		err = cmd.Run(context.Background(), []string{"mrw", "-C", root, "read", "--ast-grep", "func A"})
		return err
	})
	msg := errString(err) + out + sink.String()
	if err == nil || exitCode(err) != exitUsage || !strings.Contains(msg, "timed out") {
		t.Errorf("an ast-grep that exits 0 on TERM: exit %d, want %d and timed out:\n%s", exitCode(err), exitUsage, msg)
	}
	if strings.Contains(out, "==> a.go") {
		t.Errorf("the hit it printed on TERM was served:\n%s", out)
	}
}
