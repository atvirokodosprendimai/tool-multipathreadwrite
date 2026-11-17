//go:build windows

package apply

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// ADR-199 on Windows. Go reports a junction as an irregular file, not as a
// symlink, and a junction needs no privilege, so these run where the symlink
// cases skip. A junction is never the obstruction: one that leads to a
// directory is walked through, and one that leads to nothing is left alone,
// because the plan may make its target. A build that judged a link by the
// error Stat gave would have refused the first case here: Windows answers
// "the path was not found" for a target two levels deep, and Go reads that as
// "not a directory" too (the go-reviewer passes on #646; measured by hand on
// Windows 11 at df9b1127).
func TestAPathBelowAJunctionIsLeftToWhereItLeads(t *testing.T) {
	junction := func(t *testing.T, link, target string) {
		t.Helper()
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
			t.Fatalf("mklink /J %s %s: %v\n%s", link, target, err, out)
		}
	}
	apply := func(t *testing.T, root string, in ...Input) Result {
		t.Helper()
		for i := range in {
			in[i].Index, in[i].SrcLine = i, i+1
		}
		res, err := Apply(root, in, Options{})
		if err != nil {
			t.Fatalf("Apply returned an error: %v", err)
		}
		return res
	}
	create := func(path string) Input {
		return Input{Path: filepath.FromSlash(path), Op: "create", Body: []string{"x"}, Lines: -1}
	}

	t.Run("a junction to a two-level directory the same plan makes: applied", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "d/f.txt", "a\n")
		target := filepath.Join(root, "newdir", "sub")
		if err := os.MkdirAll(target, 0o755); err != nil {
			t.Fatal(err)
		}
		junction(t, filepath.Join(root, "d", "j"), target)
		if err := os.RemoveAll(filepath.Join(root, "newdir")); err != nil {
			t.Fatal(err)
		}
		res := apply(t, root, create("newdir/sub/y.txt"), create("d/j/x.txt"))
		if !res.Applied || res.Failed != 0 {
			t.Fatalf("the plan did not apply: %+v", res.Hunks)
		}
		if got := read(t, root, "newdir/sub/x.txt"); got != "x\n" {
			t.Errorf("newdir/sub/x.txt = %q, want %q", got, "x\n")
		}
	})

	t.Run("a missing directory under a junction to a directory: applied", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "d/real/keep.txt", "k\n")
		junction(t, filepath.Join(root, "d", "j"), filepath.Join(root, "d", "real"))
		res := apply(t, root, create("d/j/new/x.txt"))
		if !res.Applied || res.Failed != 0 {
			t.Fatalf("the plan did not apply: %+v", res.Hunks)
		}
		if got := read(t, root, "d/real/new/x.txt"); got != "x\n" {
			t.Errorf("d/real/new/x.txt = %q, want %q", got, "x\n")
		}
	})

	t.Run("a file under a junction to a directory is the obstruction", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "d/real/f", "a\n")
		junction(t, filepath.Join(root, "d", "j"), filepath.Join(root, "d", "real"))
		res := apply(t, root, create("d/j/f/a/b/x"))
		const want = "d/j/f/a/b/x cannot be written: d/j/f is a file, not a directory"
		if got := failedReason(res); got != want {
			t.Errorf("reason %q, want %q", got, want)
		}
	})
}
