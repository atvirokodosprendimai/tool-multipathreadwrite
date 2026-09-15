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

// A hanging ast-grep on PATH is killed at 2 s (ADR-058 Decision 5). Without
// the bound this test is killed by its own 3 s deadline — that is the red.
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
	case <-time.After(3 * time.Second):
		t.Fatal("ast-grep still running at 3 s; the 2 s bound never fired")
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
