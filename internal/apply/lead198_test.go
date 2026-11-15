package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-198 Decision 6. A refusal that `rooted` words about a rename
// destination or a `from=` source begins with the path as the platform spells
// it; the reason spells that start as the row does, and only that start. The
// reason separator is set to `\` where the platform's is `/`, so a name with a
// backslash is the one file `l/a\b.txt` there and a real directory on Windows;
// either way the wanted start is l/a/b.txt. The outside directory's own name
// holds a backslash (`o\x`), so a conversion that replaced it everywhere would
// change the absolute path the reason holds and fail.
func TestARefusedDestinationOrSourceBeginsAsTheRowDoes(t *testing.T) {
	prev := reasonSep
	t.Cleanup(func() { reasonSep = prev })
	if filepath.Separator == '/' {
		reasonSep = '\\'
	}
	const typed = `l/a\b.txt`
	const row = "l/a/b.txt"

	// linked makes a root whose link l leads out of it, to a directory named
	// o\x, and returns the real outside path the reason must hold.
	linked := func(t *testing.T) (root, outside string) {
		t.Helper()
		root = t.TempDir()
		write(t, root, "f0.txt", "a\nb\n")
		out := filepath.Join(t.TempDir(), `o\x`)
		if err := os.MkdirAll(out, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(out, filepath.Join(root, "l")); err != nil {
			t.Skipf("links unavailable here: %v", err)
		}
		real, err := filepath.EvalSymlinks(out)
		if err != nil {
			t.Fatal(err)
		}
		return root, real
	}
	reasonOf := func(t *testing.T, root string, in Input) string {
		t.Helper()
		in.Index, in.SrcLine = 0, 1
		res, err := Apply(root, []Input{in}, Options{})
		if err != nil {
			t.Fatal(err)
		}
		r := failedReason(res)
		if r == "" {
			t.Fatalf("no hunk failed: %+v", res.Hunks)
		}
		return r
	}

	t.Run("a rename destination", func(t *testing.T) {
		root, outside := linked(t)
		r := reasonOf(t, root, Input{Path: "f0.txt", Op: "rename", Body: []string{typed}, Lines: -1})
		if !strings.HasPrefix(r, row+" resolves to ") {
			t.Errorf("reason %q, want it to begin %q", r, row+" resolves to ")
		}
		if !strings.Contains(r, outside) {
			t.Errorf("reason %q lost the outside path %q, which is the system's and kept byte for byte", r, outside)
		}
	})
	t.Run("a from= source", func(t *testing.T) {
		root, outside := linked(t)
		r := reasonOf(t, root, Input{Path: "g.txt", Op: "create", From: typed, FromStart: 1, FromEnd: 1, Lines: -1})
		if want := "from=" + row + ": " + row + " resolves to "; !strings.HasPrefix(r, want) {
			t.Errorf("reason %q, want it to begin %q", r, want)
		}
		if !strings.Contains(r, outside) {
			t.Errorf("reason %q lost the outside path %q", r, outside)
		}
	})
	t.Run("the platform's own separator keeps a backslash in the name", func(t *testing.T) {
		if filepath.Separator != '/' {
			t.Skip("here a backslash is the separator, so there is no name to keep")
		}
		reasonSep = '/'
		root, _ := linked(t)
		r := reasonOf(t, root, Input{Path: "f0.txt", Op: "rename", Body: []string{typed}, Lines: -1})
		if !strings.HasPrefix(r, typed+" resolves to ") {
			t.Errorf("reason %q, want it to begin %q", r, typed+" resolves to ")
		}
	})

	t.Run("the function converts only an exact path at the start", func(t *testing.T) {
		p := `l/a\b.txt`
		for _, c := range []struct{ name, text, want string }{
			{"followed by a space, and the system's path kept", `l/a\b.txt resolves to /tmp/o\x/a\b.txt, which is outside`, `l/a/b.txt resolves to /tmp/o\x/a\b.txt, which is outside`},
			{"followed by a colon", `l/a\b.txt: Windows does not keep it`, `l/a/b.txt: Windows does not keep it`},
			{"the path only inside the sentence", `no line of l/a\b.txt holds it`, `no line of l/a\b.txt holds it`},
			{"a longer name", `l/a\b.txt2 resolves to x`, `l/a\b.txt2 resolves to x`},
			{"advice to read it", "Run `mrw read l/a\\b.txt` first", "Run `mrw read l/a\\b.txt` first"},
			{"another path", `l/a\c.txt resolves to x`, `l/a\c.txt resolves to x`},
		} {
			if got := SlashLead(c.text, p, '\\'); got != c.want {
				t.Errorf("%s: SlashLead(%q) = %q, want %q", c.name, c.text, got, c.want)
			}
		}
		if got := SlashLead(`l/a/b.txt resolves to x`, "l/a/b.txt", '\\'); got != `l/a/b.txt resolves to x` {
			t.Errorf("a path with no separator in it changed: %q", got)
		}
	})
}
