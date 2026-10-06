package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ADR-126. A write whose check's deadline had passed before the check started
// said "no check could run … timed out", exit 2. The write landed and nothing
// verified it, which is exit 3's meaning.
func TestAWriteWhoseDeadlinePassedBeforeItsCheckStartedExits3(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\n",
		".quality-harness.json": `{"check":"exit 0"}`,
	})
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	plan := planFile(t, "@@ a.go 2 replace\nfunc A() { _ = 2 }\n")
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	out, err := captureStdout(t, func() error {
		return rootCommand().Run(ctx, []string{"mrw", "-C", root, "write", "--check", plan})
	})
	if err == nil || exitCode(err) != exitCheckFailed || !strings.Contains(err.Error(), "timed out before it started") || strings.Contains(err.Error(), "declare one") {
		t.Fatalf("want exit %d, timed out before it started: %v\n%s", exitCheckFailed, err, out)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "a.go")); !strings.Contains(string(b), "_ = 2") {
		t.Errorf("the write did not land: %q", b)
	}
}

// ADR-126, the same for `mrw check`.
func TestMrwCheckWhoseDeadlinePassedBeforeItStartedExits3(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\n",
		".quality-harness.json": `{"check":"exit 0"}`,
	})
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	out, err := captureStdout(t, func() error {
		return rootCommand().Run(ctx, []string{"mrw", "-C", root, "check", "a.go"})
	})
	if err == nil || exitCode(err) != exitCheckFailed || !strings.Contains(err.Error(), "timed out before it started") || strings.Contains(out+err.Error(), "declare one") {
		t.Fatalf("want exit %d, timed out before it started: %v\n%s", exitCheckFailed, err, out)
	}
}

// The in-process and Codex reviews of #340. A check that could not start for
// want of a shell, under a deadline that had also passed, was called a
// timeout before it started, exit 3: the missing shell is exit 2, with its
// advice, as ADR-082 says.
func TestMrwCheckWithNoShellUnderAnExpiredDeadlineStillExits2(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\n",
		".quality-harness.json": `{"check":"exit 0"}`,
	})
	t.Setenv("PATH", "")
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	out, err := captureStdout(t, func() error {
		return rootCommand().Run(ctx, []string{"mrw", "-C", root, "check", "a.go"})
	})
	if err == nil || exitCode(err) != exitUsage || strings.Contains(err.Error(), "before it started") {
		t.Fatalf("want exit %d, could not start: %v\n%s", exitUsage, err, out)
	}
}
