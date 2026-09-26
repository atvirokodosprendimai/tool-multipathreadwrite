//go:build unix

package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// ADR-080 (the waiver on #232). An ast-grep that answered and exited 0 while a
// background grandchild of its wrapper kept running left that grandchild behind:
// the group was killed only on a timeout. The hit is served and the grandchild
// is reaped.
func TestAnAstGrepThatExitsCleanlyLeavesNoGrandchild(t *testing.T) {
	root := grepTree(t, map[string]string{"hit.go": "package hit\nfunc Target() {}\n"})
	pidFile := filepath.Join(t.TempDir(), "gc.pid")
	dir := t.TempDir()
	script := "#!/bin/sh\nsleep 30 >/dev/null 2>&1 &\necho $! > '" + pidFile + "'\n" +
		`printf '%s' '[{"file":"hit.go","range":{"start":{"line":1},"end":{"line":1}}}]'` + "\nexit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "ast-grep"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := readIn(t, root, "--ast-grep", "func Target")
	if err != nil || !strings.Contains(out, "func Target") {
		t.Fatalf("the hit was not served: %v\n%s", err, out)
	}
	b, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	deadline := time.Now().Add(3 * time.Second)
	for syscall.Kill(pid, 0) == nil {
		if time.Now().After(deadline) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
			t.Fatal("ast-grep's grandchild outlived a clean exit")
		}
		time.Sleep(50 * time.Millisecond)
	}
}
