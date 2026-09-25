package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// paddedTree holds x and "x " with different bytes, and skips where the
// filesystem cannot keep the two names apart.
func paddedTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for n, b := range map[string]string{"x": "plain\n", "x ": "padded\n"} {
		if err := os.WriteFile(filepath.Join(root, n), []byte(b), 0o644); err != nil {
			t.Skipf("this filesystem cannot hold %q: %v", n, err)
		}
	}
	if fi, err := os.ReadDir(root); err != nil || len(fi) != 2 {
		t.Skip("this filesystem folds names that differ by a trailing space")
	}
	return root
}

// ADR-069 T5, from the Codex review of v1.25.0. The guard stopped at every
// "--", including one consumed as a flag's value, so `read --grep -- 'x '`
// still let the parser trim the path to x. A flag's own value is skipped, and
// only a "--" in argument position ends the guard. The same rule retires the
// false refusal of a padded flag value whose trim equals a positional.
func TestAFlagValueDoesNotEndThePaddedArgGuard(t *testing.T) {
	root := paddedTree(t)
	out, code := runIn(t, root, "read", "--grep", "--", "x ")
	if code != exitUsage || !strings.Contains(out, "'x '") {
		t.Errorf("read --grep -- 'x ' exited %d, want %d naming 'x ':\n%s", code, exitUsage, out)
	}
	if out, code := runIn(t, root, "read", "--grep", "plain", "--exclude", " x", "x"); code != 0 {
		t.Errorf("a padded --exclude value beside the positional x was refused, exit %d:\n%s", code, out)
	}
	if out, code := runIn(t, root, "read", "--grep", "plain", "--", "x "); code != 1 || strings.Contains(out, "1| plain") {
		t.Errorf("read --grep plain -- 'x ' exited %d (want 1: x-space holds no match) or served x:\n%s", code, out)
	}
}

// ADR-069 T5. urfave trims a whole attached token, so --files-from='list '
// opened list. The value is refused, and the separate spelling, which the
// parser keeps as given, still works. Root flags never reach a subcommand's
// raw tail, so refusePaddedFlagValues checks the whole argv in main.
func TestAnAttachedFlagValueWithTrailingSpaceIsRefused(t *testing.T) {
	root := paddedTree(t)
	list := filepath.Join(t.TempDir(), "list")
	for n, b := range map[string]string{list: "x\n", list + " ": "x \n"} {
		if err := os.WriteFile(n, []byte(b), 0o644); err != nil {
			t.Skipf("cannot write %q: %v", n, err)
		}
	}
	out, code := runIn(t, root, "read", "--files-from="+list+" ")
	if code != exitUsage || !strings.Contains(out, "own argument") {
		t.Errorf("--files-from='list ' exited %d, want %d saying to pass the value as its own argument:\n%s", code, exitUsage, out)
	}
	if out, code := runIn(t, root, "read", "--files-from", list+" "); code != 0 || !strings.Contains(out, "padded") {
		t.Errorf("--files-from 'list ' exited %d or did not serve x-space:\n%s", code, out)
	}
	if err := refusePaddedFlagValues([]string{"--root=/tmp/d ", "read", "x"}); err == nil {
		t.Error("a padded attached --root value was not refused")
	}
	if err := refusePaddedFlagValues([]string{"read", "--", "--root=x "}); err != nil {
		t.Errorf("a path after -- was refused: %v", err)
	}
}

// ADR-069 T5. The iter refusal suggested `mrw iter -- 'x '`, which makes the
// path the verb; the suggestion keeps the verb.
func TestTheIterRefusalKeepsTheVerb(t *testing.T) {
	root := paddedTree(t)
	out, code := runIn(t, root, "iter", "add", "x ")
	if code != exitUsage || !strings.Contains(out, "mrw iter add -- 'x '") {
		t.Errorf("iter add 'x ' exited %d, want %d suggesting mrw iter add -- 'x ':\n%s", code, exitUsage, out)
	}
}
