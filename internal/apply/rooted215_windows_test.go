//go:build windows

package apply

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-215 on Windows. `\x`, `c:x`, `C:/x` and a forward-slash UNC path carry no
// separator-and-volume pair IsAbs accepts, so each is refused by rooted.IsRooted
// with the one sentence; the relative spellings Windows reads as something else
// keep the refusals they had. It runs on the Windows CI shard; on any other
// platform the fence only compiles it.
func TestAWindowsRootedPlanPathIsRefused(t *testing.T) {
	run := func(t *testing.T, root, path string) HunkResult {
		t.Helper()
		in := []Input{{Path: path, Op: "create", Body: []string{"x"}, Lines: -1, Index: 0, SrcLine: 1}}
		res, err := Apply(root, in, Options{})
		if err != nil || len(res.Hunks) != 1 {
			t.Fatalf("%q: %v %+v", path, err, res.Hunks)
		}
		return res.Hunks[0]
	}

	for _, p := range []string{`c:x`, `\x`, `C:/x`, `//server/share/x`, `\\server\share\x`} {
		t.Run(p, func(t *testing.T) {
			root := t.TempDir()
			h := run(t, root, p)
			if h.Status != StatusFailed || !strings.Contains(h.Reason, rootedSentence) {
				t.Errorf("%s: %s %q, want failed with %q", p, h.Status, h.Reason, rootedSentence)
			}
			if ents := snapshot215(t, root); len(ents) != 1 {
				t.Errorf("%s: the root holds %v, want it empty", p, ents)
			}
		})
	}

	// Relative spellings that meet their own refusals, not this one.
	t.Run("relative spellings keep their refusals", func(t *testing.T) {
		root, outside := t.TempDir(), t.TempDir()
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", filepath.Join(root, "j"), outside).CombinedOutput(); err != nil {
			t.Fatalf("mklink /J: %v\n%s", err, out)
		}
		for _, p := range []string{"nul.txt", "b.txt::$DATA", `j\x`} {
			h := run(t, root, p)
			if h.Status != StatusFailed || h.Reason == "" || strings.Contains(h.Reason, rootedSentence) {
				t.Errorf("%s: %s %q, want its own refusal, not the rooted one", p, h.Status, h.Reason)
			}
		}
	})
}
