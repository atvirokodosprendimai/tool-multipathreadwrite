package adversarial

import (
	"os/exec"
	"strings"
	"testing"
)

// TestNoTrackedPathIsLongerThan120 keeps a plain `git clone` working on Windows.
// Git for Windows refuses a path longer than MAX_PATH (259 characters) unless
// core.longpaths is set, and the longest task file ran to 151, so a clone
// under a root longer than about 108 characters aborted with "Filename too
// long" — found by two Windows sessions in the v1.25.1 field tests. With every
// tracked path at 120 or under, a root of up to 138 characters clones as is.
//
// It asks git for the tracked files rather than walking the checkout, so a
// long untracked path in someone's working tree is not the repository's
// problem; and it fails, rather than skipping, when git cannot answer, because
// a guard that looked at nothing would pass.
func TestNoTrackedPathIsLongerThan120(t *testing.T) {
	const budget = 120
	cmd := exec.Command("git", "ls-files", "-z")
	cmd.Dir = repoRoot(t)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}
	paths := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")
	// A floor, not a presence check: the repository tracks well over a
	// thousand files, and a listing that came back nearly empty would pass.
	if len(paths) < 500 {
		t.Fatalf("git listed only %d tracked paths; this guard would pass by looking at nothing", len(paths))
	}
	for _, p := range paths {
		if len(p) > budget {
			t.Errorf("%s is %d characters; keep every tracked path at %d or under (CONTRIBUTING, Prerequisites)", p, len(p), budget)
		}
	}
}
