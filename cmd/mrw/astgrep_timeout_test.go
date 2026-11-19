package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// A hanging ast-grep on PATH is stopped by a 2 s bound (ADR-058 Decision 5):
// on Windows its job is terminated then, on unix it is sent SIGTERM then and
// killed a second later if it ignores it. Without the bound the fake sleeps
// 30 s and this test is ended by its own 15 s deadline — that is the red. The
// deadline is a hang guard, not a measure of the bound: at 3 s it failed on a
// loaded Windows runner where the read had not yet returned (issue #687), so
// it outlasts slow scheduling and still ends well inside the hang.
func TestAHangingAstGrepTimesOut(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\n"})
	installHangingAstGrep(t)
	if os.Getenv("XDG_STATE_HOME") == "" {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
	}

	type result struct {
		out string
		err error
	}
	done := make(chan result, 1)
	started := time.Now()
	go func() {
		var sink bytes.Buffer
		cmd := rootCommand()
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		err := cmd.Run(context.Background(), []string{"mrw", "-C", root, "read", "--ast-grep", "zzz-absent"})
		done <- result{out: sink.String(), err: err}
	}()

	select {
	case <-time.After(15 * time.Second):
		t.Fatal("the read had not returned at 15 s: the bound on a hanging ast-grep did not end it")
	case got := <-done:
		elapsed := time.Since(started)
		if elapsed < 500*time.Millisecond {
			t.Fatalf("the hang was never entered (returned in %s)", elapsed)
		}
		msg := errString(got.err) + got.out
		if got.err == nil {
			t.Fatalf("hanging ast-grep exited 0:\n%s", got.out)
		}
		if code := exitCode(got.err); code != exitUsage {
			t.Errorf("hanging ast-grep exited %d, want %d:\n%s", code, exitUsage, msg)
		}
		if !strings.Contains(msg, "ast-grep") {
			t.Errorf("the reason does not name ast-grep:\n%s", msg)
		}
		if !strings.Contains(msg, "timed out") {
			t.Errorf("the reason does not say timed out:\n%s", msg)
		}
		if strings.Contains(msg, "not found") || strings.Contains(msg, "PATH") {
			t.Errorf("timeout was reported as a missing binary:\n%s", msg)
		}
		if strings.Contains(msg, "no file matched") {
			t.Errorf("timeout was reported as zero hits:\n%s", msg)
		}
	}
}

func installHangingAstGrep(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "hang.go")
	body := `package main
import "time"
func main() { time.Sleep(30 * time.Second) }
`
	if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "ast-grep")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, src)
	cmd.Env = append(os.Environ(), "GOFLAGS=")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building hanging ast-grep: %v\n%s", err, out)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
