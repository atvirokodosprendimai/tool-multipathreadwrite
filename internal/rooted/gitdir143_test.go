package rooted

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-143. A path that lands in a .git directory is refused by where it lands:
// as spelled (folded, cleaned), through a link, or because the root itself is
// inside one. A name that merely starts or ends like it is not.
func TestAPathInsideADotGitIsRefused(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{".git/config", "sub/.git/hooks/x", ".GIT/config", ".git", "a/../.git/HEAD", "x/./.git/y"} {
		if err := GitDir(root, p); err == nil || !strings.Contains(err.Error(), ".git") {
			t.Errorf("GitDir(%q) = %v, want a refusal naming .git", p, err)
		}
	}
	for _, p := range []string{".github/x", ".gitignore", "x.git/y", "git/config", "dot.git", "a.txt"} {
		if err := GitDir(root, p); err != nil {
			t.Errorf("GitDir(%q) = %v, want none: it is not a .git", p, err)
		}
	}
	if err := os.Symlink(".git", filepath.Join(root, "alias")); err == nil {
		for _, p := range []string{"alias/config", "alias/new/file", "alias"} {
			if err := GitDir(root, p); err == nil {
				t.Errorf("GitDir(%q) through a link to .git = nil, want a refusal", p)
			}
		}
	}
	inside := filepath.Join(t.TempDir(), ".git")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := GitDir(inside, "config"); err == nil {
		t.Error("GitDir under a root inside .git = nil, want a refusal")
	}
}

// ADR-143. The 8.3 short name of .git is a .git on a Windows build only; on
// another build git~1 is an ordinary name.
func TestTheShortNameOfDotGitIsMatchedOnlyWhereItExists(t *testing.T) {
	for _, c := range []struct {
		name  string
		short bool
		want  bool
	}{
		{".git", false, true}, {".GIT", false, true}, {"GIT~1", true, true}, {"git~12", true, true},
		{"GIT~1", false, false}, {"git~", true, false}, {"gitx~1", true, false}, {".gitx", true, false}, {"git", true, false},
	} {
		if got := isGitName(c.name, c.short); got != c.want {
			t.Errorf("isGitName(%q, %v) = %v, want %v", c.name, c.short, got, c.want)
		}
	}
}
