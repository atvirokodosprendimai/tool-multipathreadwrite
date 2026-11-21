//go:build !windows

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestTheStaticHookStopsWhatLeftItsProcessGroup is issue #775. scripts/static.sh
// runs cut copies of scripts/contract.sh, and contract.sh puts its runner in a
// process group of its own, so a kill of the script's group alone leaves a hung
// runner, and whatever the runner started, alive after the hook. The stand-in
// script starts such a runner the way contract.sh does (perl's setpgrp), the
// runner starts a child, and the hook is signalled while both run: neither may
// outlive it. Both ignore a hangup, as contract.sh's own tee does: the system
// hangs up a stopped group whose parent died, which would end them without the
// hook ever killing them.
func TestTheStaticHookStopsWhatLeftItsProcessGroup(t *testing.T) {
	if _, err := exec.LookPath("perl"); err != nil {
		t.Skip("perl is not on PATH; the stand-in cannot leave its process group")
	}
	h := newStaticHookRepo(t, `perl -e 'setpgrp(0, 0); exec @ARGV' sh -c 'trap "" HUP; sleep 300 & echo $! > kid.pid; echo $$ > runner.pid; wait' &
wait`)
	h.commit("second")
	in, _ := json.Marshal(map[string]any{
		"hook_event_name": "PostToolUse", "session_id": "s", "cwd": h.root,
		"tool_name": "Bash", "tool_input": map[string]any{"command": "git commit -m second"},
	})
	c := exec.Command(h.py, h.hook)
	c.Stdin, c.Env = strings.NewReader(string(in)), h.env
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- c.Wait() }()

	pid := func(name string) int {
		t.Helper()
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			if b, err := os.ReadFile(filepath.Join(h.root, name)); err == nil {
				if n, err := strconv.Atoi(strings.TrimSpace(string(b))); err == nil && n > 1 {
					return n
				}
			}
			time.Sleep(20 * time.Millisecond)
		}
		_ = c.Process.Kill()
		t.Fatalf("the stand-in never wrote %s: the fixture did not reach the runner", name)
		return 0
	}
	runner, kid := pid("runner.pid"), pid("kid.pid")
	t.Cleanup(func() {
		_ = syscall.Kill(runner, syscall.SIGKILL)
		_ = syscall.Kill(kid, syscall.SIGKILL)
	})
	if pgid, err := syscall.Getpgid(runner); err != nil || pgid != runner {
		t.Fatalf("the runner is not the leader of its own process group (pgid %d, %v): the fixture does not reach the case", pgid, err)
	}

	if err := c.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("the signalled hook did not exit 0: %v", err)
		}
	case <-time.After(20 * time.Second):
		_ = c.Process.Kill()
		t.Fatal("the signalled hook was still running after 20 s")
	}
	for name, p := range map[string]int{"the runner, in a process group of its own,": runner, "the runner's child": kid} {
		gone := false
		for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
			if err := syscall.Kill(p, 0); err == syscall.ESRCH {
				gone = true
				break
			}
		}
		if !gone {
			t.Errorf("%s (pid %d) outlived the hook", name, p)
		}
	}
}
