//go:build unix

package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// ADR-074 T1. An ast-grep behind a wrapper that leaves a grandchild holding its
// stdout defeated the 2 s bound: exec killed the wrapper and then waited for the
// pipe to close, which took the grandchild's 30 s, and the grandchild outlived
// mrw. The read returns at the bound now, and the grandchild is gone.
func TestAnAstGrepGrandchildHoldingStdoutDoesNotDefeatTheBound(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\n"})
	pidFile := filepath.Join(t.TempDir(), "gc.pid")
	installForkingAstGrep(t, pidFile)
	if os.Getenv("XDG_STATE_HOME") == "" {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
	}
	t.Cleanup(func() {
		if pid, err := pidIn(pidFile); err == nil {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	})

	type result struct {
		out string
		err error
	}
	done := make(chan result, 1)
	go func() {
		var sink bytes.Buffer
		cmd := rootCommand()
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		err := cmd.Run(context.Background(), []string{"mrw", "-C", root, "read", "--ast-grep", "zzz"})
		done <- result{out: sink.String(), err: err}
	}()
	select {
	case <-time.After(5 * time.Second):
		t.Fatal("the read is still running at 5 s: a grandchild holding stdout defeated the 2 s bound")
	case got := <-done:
		msg := errString(got.err) + got.out
		if exitCode(got.err) != exitUsage || !strings.Contains(msg, "timed out") {
			t.Fatalf("want exit %d naming a timeout, got exit %d:\n%s", exitUsage, exitCode(got.err), msg)
		}
	}
	pid, err := pidIn(pidFile)
	if err != nil {
		t.Fatalf("the wrapper never recorded its grandchild: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for syscall.Kill(pid, 0) == nil {
		if time.Now().After(deadline) {
			t.Fatalf("the grandchild %d is still alive 2 s after the read returned", pid)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// ADR-074 T1. ast-grep runs in a process group of its own, so the terminal's ^C
// reaches mrw and not ast-grep. mrw passes it on: a terminate sent to mrw while
// ast-grep runs stops ast-grep and its grandchild, and the read says why.
func TestAnInterruptStopsAstGrepAndItsGrandchild(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\n"})
	pidFile := filepath.Join(t.TempDir(), "gc.pid")
	installForkingAstGrep(t, pidFile)
	if os.Getenv("XDG_STATE_HOME") == "" {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
	}
	t.Cleanup(func() {
		if pid, err := pidIn(pidFile); err == nil {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	})

	done := make(chan error, 1)
	var sink bytes.Buffer
	go func() {
		cmd := rootCommand()
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		done <- cmd.Run(context.Background(), []string{"mrw", "-C", root, "read", "--ast-grep", "zzz"})
	}()
	// The wrapper records its grandchild only once mrw has started it, and mrw
	// listens before it starts anything, so the signal below has a listener.
	var pid int
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		var err error
		if pid, err = pidIn(pidFile); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("ast-grep never started")
		}
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case <-time.After(5 * time.Second):
		t.Fatal("the read is still running 5 s after mrw was told to stop")
	case err := <-done:
		msg := errString(err) + sink.String()
		if err == nil || !strings.Contains(msg, "interrupted") {
			t.Fatalf("want a read that says it was interrupted, got:\n%s", msg)
		}
	}
	deadline := time.Now().Add(2 * time.Second)
	for syscall.Kill(pid, 0) == nil {
		if time.Now().After(deadline) {
			t.Fatalf("the grandchild %d outlived the interrupt", pid)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// installForkingAstGrep puts on PATH an ast-grep that starts a grandchild
// holding its stdout, records the grandchild's pid, and waits for it.
func installForkingAstGrep(t *testing.T, pidFile string) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\nsleep 30 &\necho $! > '" + pidFile + "'\nwait\n"
	if err := os.WriteFile(filepath.Join(dir, "ast-grep"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func pidIn(file string) (int, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(b)))
}
