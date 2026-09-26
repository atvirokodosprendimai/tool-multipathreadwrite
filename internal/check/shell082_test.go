package check

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-082. A plain PowerShell PATH holds Git's cmd directory and not its
// usr\bin, so a check could not start there. The sh.exe Git for Windows ships
// is found beside git.exe from each place git.exe is installed; a tree with no
// shell, or with a directory where the shell would be, yields nothing.
func TestGitsShellIsFoundBesideGit(t *testing.T) {
	for gitRel, shRel := range map[string]string{
		"cmd/git.exe":         "usr/bin/sh.exe",
		"mingw64/bin/git.exe": "usr/bin/sh.exe",
		"bin/git.exe":         "usr/bin/sh.exe",
	} {
		root := t.TempDir()
		for _, rel := range []string{gitRel, shRel} {
			p := filepath.Join(root, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, nil, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		if got, want := gitShell(filepath.Join(root, filepath.FromSlash(gitRel))), filepath.Join(root, filepath.FromSlash(shRel)); got != want {
			t.Errorf("git at %s: gitShell = %q, want %q", gitRel, got, want)
		}
	}
	root := t.TempDir()
	git := filepath.Join(root, "cmd", "git.exe")
	if err := os.MkdirAll(filepath.Dir(git), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(git, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := gitShell(git); got != "" {
		t.Errorf("a Git tree with no shell gave %q", got)
	}
	if err := os.MkdirAll(filepath.Join(root, "usr", "bin", "sh.exe"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := gitShell(git); got != "" {
		t.Errorf("a directory named sh.exe was taken for a shell: %q", got)
	}
}

// ADR-082. With no shell anywhere the check could not start, and the report
// says there is no sh rather than leaving the caller with an exec error.
func TestACheckWithNoShellSaysSo(t *testing.T) {
	tempDirForLogs(t)
	t.Setenv("PATH", t.TempDir())
	res, err := Run(context.Background(), t.TempDir(), Config{Check: "exit 0", declared: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// A git.exe outside Git's own layouts — a scoop shim, a tools directory — is
	// not followed up to a shell beside it (the reviews of #247).
	for _, gitRel := range []string{"scoop/shims/git.exe", "repo/tools/git.exe"} {
		base := t.TempDir()
		for _, rel := range []string{gitRel, "scoop/bin/sh.exe", "scoop/usr/bin/sh.exe", "repo/bin/sh.exe", "repo/usr/bin/sh.exe"} {
			p := filepath.Join(base, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, nil, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		if got := gitShell(filepath.Join(base, filepath.FromSlash(gitRel))); got != "" {
			t.Errorf("git at %s, outside Git's layouts, gave %q", gitRel, got)
		}
	}
	if res.Ran || !strings.HasPrefix(res.Skipped, "could not start") || !strings.Contains(res.Skipped, "no sh on PATH") {
		t.Errorf("want could not start, naming the missing sh: %+v", res)
	}
}

// needShell skips a test whose check must start when this machine has no POSIX
// shell: no sh on PATH and, on Windows, no Git for Windows (ADR-082).
func needShell(t *testing.T) {
	t.Helper()
	if _, ok := Shell(); !ok {
		t.Skip("no POSIX shell: no sh on PATH and no Git for Windows (ADR-082)")
	}
}
