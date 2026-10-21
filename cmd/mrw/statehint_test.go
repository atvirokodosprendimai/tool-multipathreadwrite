package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/statehint"
)

// hinted fails the test unless statehint.Hint(err) keeps err's text, carries each
// of want, and keeps err's exit code.
func hinted(t *testing.T, name string, err error, want ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: the fixture did not fail", name)
	}
	got := statehint.Hint(err)
	for _, w := range append([]string{err.Error(), "mrw's state directory", "read ledger", "set XDG_STATE_HOME to a writable absolute directory"}, want...) {
		if !strings.Contains(got.Error(), w) {
			t.Errorf("%s: %q lacks %q", name, got, w)
		}
	}
	if exitCode(got) != exitCode(err) {
		t.Errorf("%s: exit %d, want %d", name, exitCode(got), exitCode(err))
	}
}

// A failure in mrw's state directory is named as mrw's, with the variable that
// moves it, whether XDG_STATE_HOME is set or not, whether the error reaches
// main bare or through exitWith, and whatever the call wrapped it in. A file
// where a directory must go fails for any user, root included. The pair: an
// error that is not about the state directory is unchanged, a refusal that
// echoes caller text naming the base among them.
func TestAStateDirectoryThatCannotBeMadeNamesTheVariableThatMovesIt(t *testing.T) {
	name := "state: data" // ": " inside a path name
	if runtime.GOOS == "windows" {
		name = "state data" // Windows refuses a colon in a name
	}
	base := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(base, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "mrw"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", base)
	_, err := state.Dir(t.TempDir())
	hinted(t, "through exitWith", exitWith(err), "under XDG_STATE_HOME="+base)

	// What seen.save returns when the ledger's rename fails: its words, then %w.
	ledger := filepath.Join(base, "mrw", "0123456789abcdef", "seen")
	renamed := &os.LinkError{Op: "rename", Old: ledger + ".tmp", New: ledger, Err: os.ErrPermission}
	hinted(t, "a ledger that could not be updated", exitWith(fmt.Errorf("the read ledger could not be updated; whatever this call already did stands: %w", renamed)))

	// What prune returns when the home itself cannot be opened: os.OpenRoot of
	// base's parent, here a file.
	file := filepath.Join(t.TempDir(), "home")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", file)
	_, err = os.OpenRoot(file)
	hinted(t, "prune opening the state home", exitWith(err))

	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".local"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	_, err = state.Dir(t.TempDir())
	hinted(t, "a parent of the default base", exitWith(err), "XDG_STATE_HOME is not set")

	b, err := state.Base()
	if err != nil {
		t.Fatal(err)
	}
	_, missing := os.ReadFile(filepath.Join(t.TempDir(), "missing"))
	if !errors.Is(missing, os.ErrNotExist) {
		t.Fatalf("the pair's fixture: %v", missing)
	}
	echoed := fmt.Errorf("unknown --format %q", "bad: open "+filepath.Join(b, "x"))
	for _, e := range []error{exitWith(missing), exitWith(echoed)} {
		if got := statehint.Hint(e); got != e {
			t.Errorf("an error that is not about the state directory was changed: %q", got)
		}
	}
}

// Every command fails through exitWith or refuseErr, never cli.Exit or refuse
// with the error's text, when it has an error to fail on: cli.Exit's error has
// no Unwrap and a flattened message has no type, so stateHint could not see a
// state-directory failure under either. A new such call fails this.
func TestNoCommandExitsWithAnErrorThroughCliExit(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, flat := range []string{"cli.Exit(err, ", "refuse(err.Error())", "Err.Error())", "cli.Exit(exitTwoLine(", "cli.Exit(ec.Error()"} {
			if strings.Contains(string(src), flat) {
				t.Errorf("%s calls %s…; use exitWith or refuseErr so the error stays unwrappable", f, flat)
			}
		}
	}
}

// runErr runs one command in-process and returns its error as main gets it.
func runErr(t *testing.T, root string, argv ...string) error {
	t.Helper()
	var sink bytes.Buffer
	var ran error
	_, _ = captureStdout(t, func() error {
		cmd := rootCommand()
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		ran = cmd.Run(context.Background(), append([]string{"mrw", "-C", root}, argv...))
		return ran
	})
	return ran
}

// A write that cannot use the state directory names it as a read does: the
// write's own refusals keep the error unwrappable (Codex, fourth review of
// #460, which found a write --create exiting 2 with no hint). Exit 2 either way.
func TestAWriteThatCannotUseTheStateDirectoryNamesIt(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{"a.txt": "one\n", "p.plan": "@@ a.txt 1 replace\nuno\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	file := filepath.Join(t.TempDir(), "state")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", file)
	for _, argv := range [][]string{
		{"write", "--dry-run", "--no-check", filepath.Join(root, "p.plan")},
		// The plan makers keep the error they were built from (ADR-164).
		{"write", "--grep", "one", "--sub", "uno", "--dry-run", "--no-check", "a.txt"},
		{"write", "--json", "--dry-run", "--no-check", "--grep", "one", "--sub", "uno", "a.txt"}, // --json re-spelled the exit and dropped the chain
		{"read", "--grep", "one", "--sub", "uno", "a.txt"},
	} {
		err := runErr(t, root, argv...)
		if err == nil {
			t.Fatalf("%v: want a failure with XDG_STATE_HOME a file", argv)
		}
		if got := statehint.Hint(err).Error(); !strings.Contains(got, "mrw's state directory") {
			t.Errorf("%v: %q does not name the state directory", argv, got)
		}
		if code := exitCode(err); code != 2 {
			t.Errorf("%v: exit %d, want 2", argv, code)
		}
	}
	// ADR-166 superseded the content read's row: a read of a.txt now serves
	// with exit 0, and says nothing was recorded (readonly166_test.go holds
	// the stderr line through the built binary).
	if err := runErr(t, root, "read", "a.txt"); err != nil {
		t.Errorf("a content read with XDG_STATE_HOME a file: %v, want served", err)
	}
}

// The landing's own failure keeps its words and its type: a directory where the
// write lock goes lets the read and the ledger load succeed, then fails the lock,
// which the human receipt reported through exitTwoLine as text (Codex, fifth
// review of #460).
func TestALandingThatCannotTakeTheWriteLockNamesTheStateDirectory(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	for name, body := range map[string]string{"a.txt": "one\n", "p.plan": "@@ a.txt 1 replace\nuno\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := runErr(t, root, "read", "a.txt"); err != nil {
		t.Fatalf("read: %v", err)
	}
	dir, err := state.Dir(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "seen.write.lock"), 0o700); err != nil {
		t.Fatal(err)
	}
	err = runErr(t, root, "write", "--no-check", filepath.Join(root, "p.plan"))
	if err == nil {
		t.Fatal("the write took a lock that is a directory")
	}
	if got := statehint.Hint(err).Error(); !strings.Contains(got, "mrw's state directory") {
		t.Errorf("%q does not name the state directory", got)
	}
	if code := exitCode(err); code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}
