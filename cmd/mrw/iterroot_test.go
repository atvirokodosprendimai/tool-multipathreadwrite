package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `iter add` is the FOURTH way into the tree, after read, write and check, and
// it was the one that did not enforce the root boundary.
//
// It validated with filepath.Join, which cleans `../outside/x` into a path that
// EXISTS, so the entry was accepted. Nothing leaked — read refuses it at serve
// time — but the working set is what `mrw check` scopes to by default, so one
// accepted entry made every later check refuse at exit 2 until someone removed
// it. ADR-006 rule 2: the boundary lives in one place, and this was the caller
// that did not use it.
//
// The same Join re-rooted an ABSOLUTE path onto the root, where it did not
// exist, so those were refused as "no such file" — the right answer for the
// wrong reason, which is why the two spellings disagreed.
func TestIterAddRefusesAPathOutsideTheRoot(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	outer := t.TempDir()
	root := filepath.Join(outer, "proj")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outer, "secret.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "in.txt"), []byte("y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A symlink out is the same escape wearing a different hat, and it is the
	// spelling a caller actually reaches for.
	if err := os.Symlink(outer, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	run := func(args ...string) (error, string) {
		t.Helper()
		cmd := rootCommand()
		var buf bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &buf, &buf
		err := cmd.Run(context.Background(), append([]string{"mrw", "-C", root}, args...))
		return err, buf.String()
	}

	for _, spec := range []string{"../secret.txt", "link/secret.txt"} {
		err, out := run("iter", "add", spec)
		if err == nil {
			t.Errorf("%s: accepted a path outside the root\n%s", spec, out)
			continue
		}
		if !strings.Contains(err.Error()+out, "outside the root") {
			t.Errorf("%s: refused for the wrong reason: %v", spec, err)
		}
	}

	// CONTROLS. A refusal that swallowed ordinary specs would be worse than the
	// bug: the working set is how a caller keeps a task in one place.
	if err, out := run("iter", "add", "in.txt"); err != nil {
		t.Errorf("an in-root file was refused: %v\n%s", err, out)
	}
	if err, out := run("iter", "add", "in.txt:1"); err != nil {
		t.Errorf("an in-root ranged spec was refused: %v\n%s", err, out)
	}
	// And a missing in-root path keeps its own message, which is a different
	// mistake with a different remedy.
	err, out := run("iter", "add", "nosuch.txt")
	if err == nil {
		t.Error("a missing file was accepted")
	} else if !strings.Contains(err.Error()+out, "no such file") {
		t.Errorf("a missing file now reports the wrong reason: %v", err)
	}
}
