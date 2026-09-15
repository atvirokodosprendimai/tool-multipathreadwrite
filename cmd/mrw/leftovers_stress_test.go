package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Named CLI/hook attacks against UC-2 and UC-4. The unit tests already cover
// the happy and failure paths; these are the shapes a hostile PATH binary or a
// hanging matcher would try.

func TestAstGrepAndFilesFromAreTwoSources(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\n"})
	list := filepath.Join(t.TempDir(), "list")
	if err := os.WriteFile(list, []byte("a.go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := readIn(t, root, "--ast-grep", "fmt.Println", "--files-from", list)
	msg := errString(err) + out
	if err == nil {
		t.Fatalf("ast-grep plus --files-from exited 0:\n%s", out)
	}
	if got := exitCode(err); got != exitUsage {
		t.Errorf("exited %d, want %d:\n%s", got, exitUsage, msg)
	}
	if !strings.Contains(msg, "two sources") {
		t.Errorf("the refusal does not say two sources:\n%s", msg)
	}
}

func TestExcludeWithAstGrepIsAllowed(t *testing.T) {
	root := grepTree(t, map[string]string{"hit.go": "package hit\nfunc Target() {}\n"})
	installFakeAstGrep(t, `[{"file":"hit.go","range":{"start":{"line":1},"end":{"line":1}}}]`, 0)
	out, err := readIn(t, root, "--ast-grep", "func Target", "--exclude", "*.go")
	msg := errString(err) + out
	if strings.Contains(msg, "without --grep") {
		t.Fatalf("--exclude with --ast-grep was refused as exclude-without-grep:\n%s", msg)
	}
	if err == nil {
		t.Fatalf("every hit excluded should be zero-hits, not success:\n%s", out)
	}
	if !strings.Contains(msg, "func Target") {
		t.Errorf("zero hits should name the pattern:\n%s", msg)
	}
}

func TestARangeAndAstGrepAreTwoAnswers(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\n"})
	out, err := readIn(t, root, "--ast-grep", "fmt.Println", "a.go:1-2")
	msg := errString(err) + out
	if err == nil {
		t.Fatalf("a range plus --ast-grep exited 0:\n%s", out)
	}
	if got := exitCode(err); got != exitUsage {
		t.Errorf("exited %d, want %d:\n%s", got, exitUsage, msg)
	}
	if !strings.Contains(msg, "two answers") {
		t.Errorf("the refusal does not say two answers:\n%s", msg)
	}
}

func TestAstGrepHostileObjectStdoutIsNotZeroHits(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\n"})
	installFakeAstGrep(t, `{"hits":[]}`, 0)
	out, err := readIn(t, root, "--ast-grep", "zzz-absent")
	msg := errString(err) + out
	if err == nil {
		t.Fatalf("object JSON exited 0:\n%s", out)
	}
	if got := exitCode(err); got == 1 && strings.Contains(msg, "no file matched") {
		t.Errorf("a hostile object must not take the zero-hits path:\n%s", msg)
	}
}

func TestAstGrepExitOneWithEmptyArrayNamesThePattern(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\n"})
	installFakeAstGrep(t, `[]`, 1)
	out, err := readIn(t, root, "--ast-grep", "zzz-absent")
	msg := errString(err) + out
	if err == nil {
		t.Fatalf("zero hits exited 0:\n%s", out)
	}
	if strings.Contains(msg, "not found") || strings.Contains(msg, "PATH") {
		t.Errorf("exit 1 plus [] was reported as a missing binary:\n%s", msg)
	}
	if !strings.Contains(msg, "zzz-absent") {
		t.Errorf("zero hits does not name the pattern:\n%s", msg)
	}
}

func TestTheRulesHookArmsTheAlarmBeforeRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGALRM is not a Windows signal")
	}
	src, err := os.ReadFile(hookFromSettings(t))
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	alarm := strings.Index(body, "signal.alarm(2)")
	run := strings.Index(body, "run(json.loads")
	if alarm < 0 {
		t.Fatal("the hook does not arm signal.alarm(2)")
	}
	if strings.Contains(body, "signal.alarm(0)") {
		t.Fatal("alarm(0) disables the bound")
	}
	if run < 0 || alarm > run {
		t.Fatal("the alarm must be armed before matching runs")
	}
	on := body[strings.Index(body, "def _on_alarm"):strings.Index(body, "def main")]
	if !strings.Contains(on, "os._exit(0)") {
		t.Fatal("_on_alarm must os._exit(0); sys.exit is not enough")
	}
}

func TestTheRulesHookHangInMatchingStillExitsZero(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGALRM is not a Windows signal")
	}
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is not on PATH; the hook cannot run here")
	}
	src, err := os.ReadFile(hookFromSettings(t))
	if err != nil {
		t.Fatal(err)
	}
	mut := strings.Replace(string(src), "def seg_match(pat, s):", "def seg_match(pat, s):\n    import time\n    time.sleep(30)\n", 1)
	if mut == string(src) {
		t.Fatal("could not inject a hang into seg_match")
	}
	path := filepath.Join(t.TempDir(), "rules-on-read.py")
	if err := os.WriteFile(path, []byte(mut), 0o644); err != nil {
		t.Fatal(err)
	}
	proj, home := t.TempDir(), t.TempDir()
	for name, body := range map[string]string{
		".claude/rules/scoped.md": "---\npaths:\n  - \"*.md\"\n---\n\nRULE\n",
		"x.md":                    "hi\n",
	} {
		full := filepath.Join(proj, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	in, _ := json.Marshal(map[string]any{
		"hook_event_name": "PostToolUse", "session_id": "s-match-hang",
		"cwd": proj, "tool_name": "Bash",
		"tool_input": map[string]any{"command": "mrw read x.md:1"},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, py, path)
	c.Stdin = strings.NewReader(string(in))
	c.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+proj, "HOME="+home, "XDG_CACHE_HOME="+filepath.Join(home, ".cache"))
	start := time.Now()
	err = c.Run()
	elapsed := time.Since(start)
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatal("a hang in matching was still running at 3 s")
	}
	if elapsed < 500*time.Millisecond {
		t.Fatalf("matching was never entered (returned in %s); the hang did not run", elapsed)
	}
	if elapsed > 2500*time.Millisecond {
		t.Fatalf("matching hang returned after %s", elapsed)
	}
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("hook run: %v", err)
		}
	}
	if code != 0 {
		t.Fatalf("matching hang exited %d, want 0", code)
	}
}
