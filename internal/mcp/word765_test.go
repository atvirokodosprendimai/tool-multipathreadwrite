package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// #765. mrw_read with grep or ast_grep names a path its walk could not use with
// the word the CLI prints for it, taken from the same Problem.Unreadable flag:
// UNREADABLE for an I/O failure (a path that is not there, a directory that
// cannot be listed), REFUSED for mrw's own rule (a path outside the root). The
// line keeps its `-- PATH: ` lead, so a reader that matched `-- PATH:` still
// does; the word follows it. Asked both where nothing matched and beside a
// served file, under each finder.
func TestAPlainMCPFinderNamesTheKindOfAPathItCouldNotUse(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("a mode-000 directory is still listable here")
	}
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "state"))
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\nvar WANTED = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "locked"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "locked", "b.go"), []byte("package b\nvar WANTED = 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(root, "locked"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, "locked"), 0o755) })
	outside := "../outside-" + filepath.Base(root)
	bad := []any{"absent.go", outside, "locked"}
	want := map[string]string{"absent.go": "UNREADABLE", outside: "REFUSED", "locked": "UNREADABLE"}
	check := func(t *testing.T, text string) {
		t.Helper()
		for path, word := range want {
			other := "REFUSED"
			if word == "REFUSED" {
				other = "UNREADABLE"
			}
			if !strings.Contains(text, "\n-- "+path+": "+word+" ") || strings.Contains(text, "-- "+path+": "+other) {
				t.Errorf("%s is not named %s:\n%s", path, word, text)
			}
		}
	}
	for _, finder := range []string{"grep", "ast_grep"} {
		t.Run(finder+"_no_match", func(t *testing.T) {
			if finder == "ast_grep" {
				installFakeAstGrep(t, "[]")
			}
			res := call(t, root, "mrw_read", map[string]any{"specs": bad, finder: "NOWHERE765"})
			check(t, fmt.Sprint(res["content"]))
		})
		t.Run(finder+"_beside_a_served_file", func(t *testing.T) {
			if finder == "ast_grep" {
				installFakeAstGrep(t, `[{"file":"a.go","range":{"start":{"line":1},"end":{"line":1}}}]`)
			}
			res := call(t, root, "mrw_read", map[string]any{"specs": append([]any{"a.go"}, bad...), finder: "WANTED"})
			text := fmt.Sprint(res["content"])
			if !strings.Contains(text, "WANTED = 1") {
				t.Fatalf("a.go was not served, so this is not the served branch:\n%s", text)
			}
			check(t, text)
		})
	}
}

// #765. An index (a grep too large to serve) names a path its walk could not use
// with the same word as a served answer does.
func TestAnIndexNamesTheKindOfAPathItCouldNotUse(t *testing.T) {
	root, args, r, e, _ := bands(t)
	c := r + (e-r)/2
	if !(r < c && c < e) {
		t.Fatalf("no ceiling separates the report (%d) from the encoded answer (%d)", r, e)
	}
	withCeiling(t, c)
	text := indexOf(t, call(t, root, "mrw_read", args), fmt.Sprintf("a read whose receipt overflows a %d ceiling", c))
	if !strings.Contains(text, "-- no-such-dir: UNREADABLE ") {
		t.Errorf("the index does not name the missing path UNREADABLE:\n%.800s", text)
	}
}

// #765. MCP grep says UNREADABLE of a file spelled as a directory (b.txt/), as
// the CLI and a direct read do (ADR-076 Decision 1).
func TestAPlainMCPFinderSaysUnreadableOfAFileSpelledAsADirectory(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "state"))
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	text := fmt.Sprint(call(t, root, "mrw_read", map[string]any{"specs": []any{"b.txt/"}, "grep": "foo"})["content"])
	if !strings.Contains(text, "\n-- b.txt/: UNREADABLE ") {
		t.Errorf("b.txt/ is not named UNREADABLE:\n%s", text)
	}
}

// #765. A path a finder could not use is printed as the CLI prints it: quoted
// when it holds a control byte, its reason on one line (read.ShownPath and
// read.OneLine, #740). Raw, a name holding a line feed forged a second `-- `
// line in the answer. Asked of the no-match answer and of the index.
func TestAPlainMCPFinderQuotesAPathThatHoldsALineFeed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a line feed is not a legal Windows file name")
	}
	// No colon in the name: a spec's colon starts its address.
	const name = "absent\n-- other.txt REFUSED x"
	quoted := `-- "absent\n-- other.txt REFUSED x": UNREADABLE `
	forged := func(t *testing.T, text string) {
		t.Helper()
		for _, l := range strings.Split(text, "\n") {
			if strings.HasPrefix(l, "-- other.txt") {
				t.Errorf("a second line was forged from the name: %q in\n%s", l, text)
			}
		}
	}
	t.Run("no_match", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "state"))
		text := fmt.Sprint(call(t, root, "mrw_read", map[string]any{"specs": []any{name}, "grep": "NOWHERE765"})["content"])
		if !strings.Contains(text, quoted) {
			t.Errorf("the name is not quoted on one line, want %q in:\n%s", quoted, text)
		}
		forged(t, text)
	})
	t.Run("index", func(t *testing.T) {
		root, args, r, e, _ := bands(t)
		args["specs"] = append(args["specs"].([]any), name)
		// The middle of the receipt band, as TestAnIndexFromAnOverflowingReceiptNamesTheProblem
		// takes it; one more short problem line does not leave the band.
		c := r + (e-r)/2
		withCeiling(t, c)
		text := indexOf(t, call(t, root, "mrw_read", args), fmt.Sprintf("a read whose receipt overflows a %d ceiling", c))
		if !strings.Contains(text, quoted) {
			t.Errorf("the index does not quote the name on one line, want %q in:\n%.1200s", quoted, text)
		}
		forged(t, text)
	})
}
