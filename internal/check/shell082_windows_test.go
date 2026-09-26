//go:build windows

package check

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-082 end to end. With sh taken off PATH and only Git's cmd directory left
// — a plain PowerShell PATH — a declared check runs under Git's own sh, with
// Git's tools first on its PATH, so a nested sh and cat resolve; with no git
// either, the report says to install Git for Windows. Git is located here by
// git --exec-path, not by the code under test, and a missing prerequisite
// fails rather than skips: a skip here cannot be told from a pass (ci.yml, the
// reviews of #247).
func TestACheckRunsUnderGitsShellWhenShIsNotOnPath(t *testing.T) {
	out, err := exec.Command("git", "--exec-path").Output()
	if err != nil {
		t.Fatalf("this runner has no Git for Windows: git --exec-path: %v", err)
	}
	// <root>\mingw64\libexec\git-core
	root := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Clean(strings.TrimSpace(string(out))))))
	cmdDir := filepath.Join(root, "cmd")
	if _, err := os.Stat(filepath.Join(cmdDir, "git.exe")); err != nil {
		t.Fatalf("no git.exe in Git's cmd directory %s: %v", cmdDir, err)
	}
	tempDirForLogs(t)
	t.Setenv("PATH", cmdDir)
	if p, err := exec.LookPath("sh"); err == nil {
		t.Fatalf("Git's cmd directory holds an sh (%s); the fixture needs a PATH without one", p)
	}
	res, err := Run(context.Background(), t.TempDir(), Config{Check: "echo nested | sh -c cat; exit 3", declared: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ran || res.ExitCode != 3 || !strings.Contains(strings.Join(res.Tail, "\n"), "nested") {
		t.Fatalf("want the check to run under Git's sh, a nested sh and cat resolving, exit 3: %+v", res)
	}
	t.Setenv("PATH", t.TempDir())
	res, err = Run(context.Background(), t.TempDir(), Config{Check: "exit 0", declared: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ran || !strings.Contains(res.Skipped, "Git for Windows") {
		t.Errorf("with no sh and no git, want the report to name Git for Windows: %+v", res)
	}
}
