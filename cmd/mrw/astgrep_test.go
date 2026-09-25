package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestMissingAstGrepIsUsageAndNamesTheBinary is UC-2's missing-binary path:
// the flag exists, and the refusal is that the CLI is not on PATH — not that
// the flag is unknown.
func TestMissingAstGrepIsUsageAndNamesTheBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	root := grepTree(t, map[string]string{"a.go": "package a\n"})
	out, err := readIn(t, root, "--ast-grep", "fmt.Println")
	msg := errString(err) + out
	if err == nil {
		t.Fatalf("missing ast-grep exited 0:\n%s", out)
	}
	if got := exitCode(err); got != exitUsage {
		t.Errorf("missing ast-grep exited %d, want %d:\n%s", got, exitUsage, msg)
	}
	if strings.Contains(msg, "flag provided") || strings.Contains(msg, "unknown flag") {
		t.Errorf("this is an unknown-flag refusal, not a missing-binary one:\n%s", msg)
	}
	if !strings.Contains(msg, "ast-grep") {
		t.Errorf("the reason does not name ast-grep:\n%s", msg)
	}
}

// TestGrepAndAstGrepTogetherAreUsage: two finders are two sources of specs.
func TestGrepAndAstGrepTogetherAreUsage(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\n"})
	out, err := readIn(t, root, "--grep", "package", "--ast-grep", "fmt.Println")
	msg := errString(err) + out
	if err == nil {
		t.Fatalf("both find flags exited 0:\n%s", out)
	}
	if got := exitCode(err); got != exitUsage {
		t.Errorf("both find flags exited %d, want %d:\n%s", got, exitUsage, msg)
	}
	if !strings.Contains(msg, "two sources") {
		t.Errorf("the refusal does not say two sources of specs:\n%s", msg)
	}
}

// TestAstGrepServesRangesThroughRead: a present binary's hits become the
// ranges read.Run already serves. A fake CLI prints one ast-grep JSON hit.
func TestAstGrepServesRangesThroughRead(t *testing.T) {
	root := grepTree(t, map[string]string{"hit.go": "package hit\nfunc Target() {}\n"})
	installFakeAstGrep(t, `[{"file":"hit.go","range":{"start":{"line":1},"end":{"line":1}}}]`, 0)
	out, err := readIn(t, root, "--ast-grep", "func Target")
	if err != nil {
		t.Fatalf("--ast-grep with a present binary failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "hit.go") || !strings.Contains(out, "func Target") {
		t.Errorf("the hit was not served through read:\n%s", out)
	}
}

// TestAPresentAstGrepWithZeroHitsIsNotTheMissingBinaryPath: empty hits name
// the pattern, like --grep, and do not claim the CLI is absent.
func TestAPresentAstGrepWithZeroHitsIsNotTheMissingBinaryPath(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\n"})
	installFakeAstGrep(t, `[]`, 0)
	out, err := readIn(t, root, "--ast-grep", "zzz-absent")
	msg := errString(err) + out
	if err == nil {
		t.Fatalf("zero hits exited 0:\n%s", out)
	}
	if strings.Contains(msg, "not found") || strings.Contains(msg, "PATH") {
		t.Errorf("zero hits was reported as a missing binary:\n%s", msg)
	}
	if !strings.Contains(msg, "zzz-absent") {
		t.Errorf("zero hits does not name the pattern:\n%s", msg)
	}
}

// TestAstGrepObservesOnlyServedLines: license is the lines read.Run printed,
// not AST nodes and not files the finder merely opened.
func TestAstGrepObservesOnlyServedLines(t *testing.T) {
	root := grepTree(t, map[string]string{
		"hit.go":  "package hit\nfunc Target() {}\n",
		"miss.go": "package miss\n",
	})
	installFakeAstGrep(t, `[{"file":"hit.go","range":{"start":{"line":1},"end":{"line":1}}}]`, 0)
	out, err := readIn(t, root, "--ast-grep", "func Target")
	if err != nil {
		t.Fatalf("--ast-grep failed: %v\n%s", err, out)
	}
	if strings.Contains(out, "miss.go") {
		t.Errorf("a file with no hit was served:\n%s", out)
	}
	if !strings.Contains(out, "@@ 2-2") && !strings.Contains(out, "@@ 2") {
		t.Errorf("the served span is not the hit's line range:\n%s", out)
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func installFakeAstGrep(t *testing.T, stdout string, exit int) {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "fake.go")
	body := fmt.Sprintf("package main\nimport (\"fmt\"; \"os\")\nfunc main() { fmt.Fprint(os.Stdout, %q); os.Exit(%d) }\n", stdout, exit)
	if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "ast-grep")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, src)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fake ast-grep: %v\n%s", err, out)
	}
	// Run it once, untimed, before any test times it. The first start of a
	// freshly built .exe on a Windows runner (a Defender scan) took long enough
	// to hit the 2 s ast-grep bound: CI 2026-09-25, 4.55 s, "zero hits does not
	// name the pattern" because the answer was "timed out". The bound is a
	// promise and stays; the fixture must not spend it.
	_ = exec.Command(bin).Run()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// ADR-064: the CLI serves a named file the glob would drop through --ast-grep,
// the same answer --grep gives.
func TestANamedFileIsServedThroughAstGrepDespiteExclude(t *testing.T) {
	root := grepTree(t, map[string]string{"b.go": "package b\nfunc D() {}\n"})
	installFakeAstGrep(t, `[{"file":"b.go","range":{"start":{"line":1},"end":{"line":1}}}]`, 0)
	out, err := readIn(t, root, "--ast-grep", "D", "--exclude", "b.go", "b.go")
	if err != nil {
		t.Fatalf("a named excluded file must be served: %v\n%s", err, out)
	}
	if !strings.Contains(out, "==> b.go") {
		t.Fatalf("the named file was pruned:\n%s", out)
	}
}
