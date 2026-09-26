package main

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// ADR-080. A timed-out check kept its log and the report named nowhere to find
// it; it names it, and the file is there.
func TestATimedOutCheckNamesItsLog(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the check runs through sh")
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("TMPDIR", t.TempDir())
	root := grepTree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\n",
		".quality-harness.json": `{"check":"echo started; sleep 30","timeout_seconds":1}`,
	})
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	out, code := runIn(t, root, "write", planFile(t, "@@ a.go 2 replace\nfunc A() { _ = 1 }\n"))
	if code != exitCheckFailed {
		t.Fatalf("exit %d, want %d:\n%s", code, exitCheckFailed, out)
	}
	m := regexp.MustCompile(`timed out after [^\n]* — full output: (\S+)`).FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("the timed-out check does not name its log:\n%s", out)
	}
	if _, err := os.Stat(m[1]); err != nil {
		t.Errorf("the named log is not there: %v", err)
	}
}

// ADR-080. A write whose check was cancelled before it started said "no check
// could run … declare one", exit 2. It says interrupted, exit 3 — the tree is
// changed and unverified, as when the check is stopped mid-run.
func TestAWriteWhoseCheckIsCancelledBeforeItStartsSaysInterrupted(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\n",
		".quality-harness.json": `{"check":"exit 0"}`,
	})
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	plan := planFile(t, "@@ a.go 2 replace\nfunc A() { _ = 2 }\n")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out, err := captureStdout(t, func() error {
		return rootCommand().Run(ctx, []string{"mrw", "-C", root, "write", "--check", plan})
	})
	if err == nil || exitCode(err) != exitCheckFailed || !strings.Contains(err.Error(), "interrupted before it started") || strings.Contains(err.Error(), "declare one") {
		t.Fatalf("want exit %d, interrupted before it started: %v\n%s", exitCheckFailed, err, out)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "a.go")); !strings.Contains(string(b), "_ = 2") {
		t.Errorf("the write did not land: %q", b)
	}
}

// ADR-080, found by its own mutation run: only the write's path was driven, so
// `mrw check` could say "could not start … declare one" for a check cancelled
// before it started. It says interrupted, exit 3, as the write does.
func TestMrwCheckCancelledBeforeItStartsSaysInterrupted(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\n",
		".quality-harness.json": `{"check":"exit 0"}`,
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out, err := captureStdout(t, func() error {
		return rootCommand().Run(ctx, []string{"mrw", "-C", root, "check", "a.go"})
	})
	if err == nil || exitCode(err) != exitCheckFailed || !strings.Contains(err.Error(), "interrupted before it started") || strings.Contains(out+err.Error(), "declare one") {
		t.Fatalf("want exit %d, interrupted before it started: %v\n%s", exitCheckFailed, err, out)
	}
}
