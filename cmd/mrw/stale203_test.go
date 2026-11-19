package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-203 T1, through the CLI's own commands. f.txt holds L1 to L10, is read
// whole and is written by mrw with one line NEW after line 3; the second plan
// then names a line by number alone, with no read between unless the row says so.
func TestAStaleNumberIsRefusedByTheBinary(t *testing.T) {
	const seeded = "L1\nL2\nL3\nNEW\nL4\nL5\nL6\nL7\nL8\nL9\nL10\n"
	const noShowing = "has no recorded showing in this version, and a line number alone cannot say which line you mean. Read it whole ("
	seed := func(t *testing.T) string {
		t.Helper()
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := t.TempDir()
		writeFiles(t, root, map[string]string{
			"f.txt": "L1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\nL9\nL10\n",
			"g.txt": "G1\nG2\n",
		})
		if out, code := runIn(t, root, "read", "f.txt", "g.txt"); code != 0 {
			t.Fatalf("read: exit %d:\n%s", code, out)
		}
		if out, code := writeIn(t, root, "--no-check", planFile(t, "@@ f.txt 3 insert-after\nNEW\n")); code != 0 {
			t.Fatalf("the seed write: exit %d:\n%s", code, out)
		}
		if got := cat(t, root, "f.txt"); got != seeded {
			t.Fatalf("the seed is %q", got)
		}
		return root
	}
	write := func(t *testing.T, root, doc string, flags ...string) (string, int) {
		t.Helper()
		return writeIn(t, root, append([]string{"--no-check"}, append(flags, planFile(t, doc))...)...)
	}
	read := func(t *testing.T, root string, args ...string) {
		t.Helper()
		if out, code := runIn(t, root, append([]string{"read"}, args...)...); code != 0 {
			t.Fatalf("read %v: exit %d:\n%s", args, code, out)
		}
	}
	refused := func(t *testing.T, root, doc, line string) {
		t.Helper()
		out, code := write(t, root, doc)
		if code != 1 || !strings.Contains(out, "line "+line+" of f.txt "+noShowing+"mrw read 'f.txt:"+line+"'; a --quiet read, or a line --max-cols cut, shows nothing), address it by pattern, or give anchor=") {
			t.Fatalf("want exit 1 naming line %s, got %d:\n%s", line, code, out)
		}
		if got := cat(t, root, "f.txt"); got != seeded {
			t.Fatalf("a refused plan changed the file:\n%s", got)
		}
	}
	lands := func(t *testing.T, root, doc, want string, flags ...string) {
		t.Helper()
		if out, code := write(t, root, doc, flags...); code != 0 {
			t.Fatalf("want a landing, got exit %d:\n%s", code, out)
		}
		if want != "" {
			if got := cat(t, root, "f.txt"); got != want {
				t.Fatalf("landed on the wrong line:\n%s", got)
			}
		}
	}
	onL7 := strings.Replace(seeded, "L7\n", "X\n", 1)
	on8 := "L1\nL2\nL3\nNEW\nL4\nL5\nL6\nX\nL8\nL9\nL10\n" // what a stale 8 would have hit

	t.Run("a number alone is refused and the file is unchanged", func(t *testing.T) {
		refused(t, seed(t), "@@ f.txt 8 replace\nX\n", "8")
	})
	t.Run("a range names its first line", func(t *testing.T) {
		refused(t, seed(t), "@@ f.txt 4-6 delete\n", "4")
	})
	t.Run("an insert names its line", func(t *testing.T) {
		refused(t, seed(t), "@@ f.txt 5 insert-after\nX\n", "5")
	})
	t.Run("anchor= lands on the line that holds L7", func(t *testing.T) {
		lands(t, seed(t), "@@ f.txt 8 replace anchor=\"L7\"\nX\n", onL7)
	})
	t.Run("a pattern lands", func(t *testing.T) {
		lands(t, seed(t), "@@ f.txt /^L7$/ replace\nX\n", onL7)
	})
	t.Run("a read that shows the line, then the number, lands", func(t *testing.T) {
		root := seed(t)
		read(t, root, "f.txt:8")
		lands(t, root, "@@ f.txt 8 replace\nX\n", on8)
	})
	t.Run("a read short of the range is refused, and the read of the range lands", func(t *testing.T) {
		root := seed(t)
		read(t, root, "f.txt:5-6")
		refused(t, root, "@@ f.txt 4-6 delete\n", "4")
		read(t, root, "f.txt:4-6")
		lands(t, root, "@@ f.txt 4-6 delete\n", "L1\nL2\nL3\nL6\nL7\nL8\nL9\nL10\n")
	})
	t.Run("a range with only its middle line missing names that line", func(t *testing.T) {
		root := seed(t)
		read(t, root, "f.txt:3", "f.txt:6")
		refused(t, root, "@@ f.txt 4-6 delete\n", "5")
	})
	t.Run("a quiet read shows nothing, whole or ranged, on a file mrw wrote", func(t *testing.T) {
		root := seed(t)
		read(t, root, "--quiet", "f.txt")
		refused(t, root, "@@ f.txt 8 replace\nX\n", "8")
		root = seed(t)
		read(t, root, "--quiet", "f.txt:8")
		refused(t, root, "@@ f.txt 8 replace\nX\n", "8")
	})
	t.Run("a whole quiet read of a file mrw never wrote shows nothing: a number alone is refused", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := t.TempDir()
		writeFiles(t, root, map[string]string{"f.txt": "a\nb\nc\n"})
		read(t, root, "--quiet", "f.txt")
		out, code := write(t, root, "@@ f.txt 2 replace\nB\n")
		if code != 1 || !strings.Contains(out, "line 2 of f.txt "+noShowing) || cat(t, root, "f.txt") != "a\nb\nc\n" {
			t.Fatalf("exit %d:\n%s", code, out)
		}
	})
	t.Run("a ranged quiet read of a file mrw never wrote is the per-line licence, as before", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := t.TempDir()
		writeFiles(t, root, map[string]string{"f.txt": "a\nb\nc\n"})
		read(t, root, "--quiet", "f.txt:2")
		if out, code := write(t, root, "@@ f.txt 2 replace\nB\n"); code != 0 || cat(t, root, "f.txt") != "a\nB\nc\n" {
			t.Fatalf("exit %d:\n%s", code, out)
		}
	})
	t.Run("a whole read after the write lands by number", func(t *testing.T) {
		root := seed(t)
		read(t, root, "f.txt")
		lands(t, root, "@@ f.txt 8 replace\nX\n", on8)
	})
	t.Run("force lands", func(t *testing.T) {
		lands(t, seed(t), "@@ f.txt 8 replace\nX\n", on8, "--force")
	})
	t.Run("a plan mrw computed lands", func(t *testing.T) {
		root := seed(t)
		if out, code := writeIn(t, root, "--no-check", "--grep", "^L7$", "--sub", "X", "f.txt"); code != 0 || cat(t, root, "f.txt") != onL7 {
			t.Fatalf("write --grep --sub: exit %d:\n%s", code, out)
		}
	})
	for _, c := range []struct{ plan, want string }{
		{"@@ f.txt $ replace\nX\n", "L1\nL2\nL3\nNEW\nL4\nL5\nL6\nL7\nL8\nL9\nX\n"},
		{"@@ f.txt 0 insert-after\nX\n", "X\n" + seeded},
		{"@@ f.txt 1 insert-before\nX\n", "X\n" + seeded},
		{"@@ f.txt 12 insert-before\nX\n", seeded + "X\n"},
	} {
		t.Run("a position lands: "+strings.SplitN(c.plan, "\n", 2)[0], func(t *testing.T) {
			lands(t, seed(t), c.plan, c.want)
		})
	}
	t.Run("insert-after line 1 names that line and is refused", func(t *testing.T) {
		refused(t, seed(t), "@@ f.txt 1 insert-after\nX\n", "1")
	})
	t.Run("a refused hunk skips its valid sibling on another file", func(t *testing.T) {
		root := seed(t)
		out, code := write(t, root, "@@ f.txt 8 replace\nX\n@@ g.txt 1 replace anchor=\"G1\"\nGG\n")
		if code != 1 || !strings.Contains(out, "skip") || cat(t, root, "f.txt") != seeded || cat(t, root, "g.txt") != "G1\nG2\n" {
			t.Fatalf("exit %d, want 1 and nothing written to either file:\n%s", code, out)
		}
	})
	t.Run("a file read whole and never written lands by number", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := t.TempDir()
		writeFiles(t, root, map[string]string{"f.txt": "a\nb\nc\n"})
		read(t, root, "f.txt")
		if out, code := write(t, root, "@@ f.txt 2 replace\nB\n"); code != 0 {
			t.Fatalf("exit %d:\n%s", code, out)
		}
	})
	t.Run("mrw write --create, then a numbered edit is refused, and lands with anchor=", func(t *testing.T) {
		root, run := subRun(t)
		if out, e, c := run("c1\nc2\nc3\nc4\nc5\n", "write", "--no-check", "--create", "c.txt"); c != 0 {
			t.Fatalf("create: exit %d:\n%s%s", c, out, e)
		}
		out, e, c := run("@@ c.txt 4 replace\nX\n", "write", "--no-check", "-")
		if c != 1 || !strings.Contains(out+e, "line 4 of c.txt "+noShowing) || cat(t, root, "c.txt") != "c1\nc2\nc3\nc4\nc5\n" {
			t.Fatalf("exit %d:\n%s%s", c, out, e)
		}
		if out, e, c := run("@@ c.txt 4 replace anchor=\"c4\"\nX\n", "write", "--no-check", "-"); c != 0 || cat(t, root, "c.txt") != "c1\nc2\nc3\nX\nc5\n" {
			t.Fatalf("anchored: exit %d:\n%s%s", c, out, e)
		}
	})
	t.Run("a rename's destination is refused until it is read under its name", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := t.TempDir()
		writeFiles(t, root, map[string]string{"f.txt": "a\nb\nc\n"})
		read(t, root, "f.txt")
		if out, code := write(t, root, "@@ f.txt - rename\nh.txt\n"); code != 0 {
			t.Fatalf("rename: exit %d:\n%s", code, out)
		}
		if out, code := write(t, root, "@@ h.txt 2 replace\nB\n"); code != 1 || !strings.Contains(out, "line 2 of h.txt "+noShowing) {
			t.Fatalf("exit %d:\n%s", code, out)
		}
		read(t, root, "h.txt:2")
		if out, code := write(t, root, "@@ h.txt 2 replace\nB\n"); code != 0 {
			t.Fatalf("after the read: exit %d:\n%s", code, out)
		}
	})
	t.Run("the plan read --sub prints lands when written later", func(t *testing.T) {
		root := seed(t)
		plan, code := runIn(t, root, "read", "--grep", "^L7$", "--sub", "X", "f.txt")
		if code != 0 || !strings.Contains(plan, "anchor=") {
			t.Fatalf("read --sub: exit %d:\n%s", code, plan)
		}
		lands(t, root, plan, onL7)
	})
}

func cat(t *testing.T, root, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// ADR-203 Decision 1: a line --max-cols served cut was not shown, so a number
// alone on it is refused in a file mrw wrote, and lands after a read that
// serves it whole.
func TestAStaleNumberOnALineMaxColsCutIsRefusedUntilItIsReadWhole(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	long := strings.Repeat("y", 600)
	writeFiles(t, root, map[string]string{"f.txt": "a\n" + long + "\nc\n"})
	if out, code := runIn(t, root, "read", "f.txt"); code != 0 {
		t.Fatalf("read: exit %d\n%s", code, out)
	}
	if out, code := writeIn(t, root, "--no-check", planFile(t, "@@ f.txt 1 insert-after\nNEW\n")); code != 0 {
		t.Fatalf("seed: exit %d\n%s", code, out)
	}
	t.Setenv("MRW_MAX_COLS", "200")
	if out, code := runIn(t, root, "read", "f.txt"); code != 0 || !strings.Contains(out, "[cols ") {
		t.Fatalf("read under MRW_MAX_COLS=200: exit %d, want the long line cut:\n%.300s", code, out)
	}
	const plan = "@@ f.txt 3 replace\nX\n"
	out, code := writeIn(t, root, "--no-check", planFile(t, plan))
	if code != 1 || !strings.Contains(out, "line 3 of f.txt has no recorded showing in this version") || !strings.Contains(out, "a line --max-cols cut, shows nothing") {
		t.Fatalf("the cut line: exit %d, want 1 and the reason:\n%.400s", code, out)
	}
	if out, code := runIn(t, root, "read", "--max-cols", "0", "f.txt"); code != 0 || strings.Contains(out, "[cols ") {
		t.Fatalf("read --max-cols 0: exit %d\n%.200s", code, out)
	}
	if out, code := writeIn(t, root, "--no-check", planFile(t, plan)); code != 0 || !strings.Contains(cat(t, root, "f.txt"), "\nX\nc\n") {
		t.Fatalf("after the whole read: exit %d\n%.300s", code, out)
	}
}
