package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// hinted fails the test unless stateHint(err) keeps err's text, carries each
// of want, and keeps err's exit code.
func hinted(t *testing.T, name string, err error, want ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: the fixture did not fail", name)
	}
	got := stateHint(err)
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
// main bare or through cli.Exit (whose error has no Unwrap), and whatever the
// call prefixed to it. A file where a directory must go fails for any user,
// root included. The pair: an error outside the state directory is unchanged.
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
	hinted(t, "mkdir under the base", err, "under XDG_STATE_HOME="+base)
	hinted(t, "through cli.Exit", cli.Exit(err, 2), "under XDG_STATE_HOME="+base)

	// What seen.save says when the ledger's rename fails: a prefix, then the error.
	ledger := filepath.Join(base, "mrw", "0123456789abcdef", "seen")
	renamed := &os.LinkError{Op: "rename", Old: ledger + ".tmp", New: ledger, Err: os.ErrPermission}
	hinted(t, "a ledger that could not be updated", cli.Exit(errors.New("the read ledger could not be updated; whatever this call already did stands: "+renamed.Error()), 2))

	// What prune says when the home itself cannot be opened: os.OpenRoot of
	// base's parent, here a file.
	file := filepath.Join(t.TempDir(), "home")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", file)
	_, err = os.OpenRoot(file)
	hinted(t, "prune opening the state home", cli.Exit(err, 2))

	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".local"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	_, err = state.Dir(t.TempDir())
	hinted(t, "a parent of the default base", cli.Exit(err, 2), "XDG_STATE_HOME is not set")

	_, other := os.ReadFile(filepath.Join(t.TempDir(), "missing"))
	if !errors.Is(other, os.ErrNotExist) {
		t.Fatalf("the pair's fixture: %v", other)
	}
	if got := stateHint(other); got != other {
		t.Errorf("an error outside the state directory was changed: %q", got)
	}
	if got := stateHint(cli.Exit(other, 2)); got.Error() != other.Error() {
		t.Errorf("an error outside the state directory was changed through cli.Exit: %q", got)
	}
	b, err := state.Base()
	if err != nil {
		t.Fatal(err)
	}
	echoed := cli.Exit(fmt.Errorf("unknown --format %q", "open "+filepath.Join(b, "x")), 2)
	if got := stateHint(echoed); got.Error() != echoed.Error() {
		t.Errorf("a refusal echoing the caller's text was given the hint: %q", got)
	}
}
