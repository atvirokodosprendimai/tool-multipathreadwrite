//go:build windows

package check

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-082 end to end. With sh taken off PATH and only Git's cmd directory left
// — a plain PowerShell PATH — a declared check runs under Git's own sh.exe;
// with no git either, the report says to install Git for Windows.
func TestACheckRunsUnderGitsShellWhenShIsNotOnPath(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git on this runner")
	}
	if gitShell(git) == "" {
		t.Skipf("%s is not Git for Windows", git)
	}
	tempDirForLogs(t)
	t.Setenv("PATH", filepath.Dir(git))
	if p, err := exec.LookPath("sh"); err == nil {
		t.Skipf("sh is still on PATH at %s", p)
	}
	res, err := Run(context.Background(), t.TempDir(), Config{Check: "echo from-git-sh", declared: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ran || !res.OK() {
		t.Fatalf("want the check to run under Git's sh: %+v", res)
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
