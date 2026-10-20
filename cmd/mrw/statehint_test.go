package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// A state directory mrw cannot make is named as mrw's, with the variable that
// moves it, whether XDG_STATE_HOME is set or not. A file where a directory must
// go fails the mkdir for any user, root included. The pair: a filesystem error
// outside the state base is left as it was.
func TestAStateDirectoryThatCannotBeMadeNamesTheVariableThatMovesIt(t *testing.T) {
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "mrw"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", base)
	_, err := state.Dir(t.TempDir())
	if err == nil {
		t.Fatal("Dir made a state directory under a file")
	}
	got := stateHint(err).Error()
	for _, want := range []string{err.Error(), "state directory", "read ledger", "under XDG_STATE_HOME=" + base, "set XDG_STATE_HOME to a writable absolute directory"} {
		if !strings.Contains(got, want) {
			t.Errorf("with XDG_STATE_HOME set: %q lacks %q", got, want)
		}
	}

	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".local"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	_, err = state.Dir(t.TempDir())
	if err == nil {
		t.Fatal("Dir made a state directory under a file")
	}
	got = stateHint(err).Error()
	for _, want := range []string{"state directory", "XDG_STATE_HOME is not set", "set XDG_STATE_HOME to a writable absolute directory"} {
		if !strings.Contains(got, want) {
			t.Errorf("with XDG_STATE_HOME unset: %q lacks %q", got, want)
		}
	}

	_, other := os.ReadFile(filepath.Join(t.TempDir(), "missing"))
	if !errors.Is(other, os.ErrNotExist) {
		t.Fatalf("the pair's fixture: %v", other)
	}
	if got := stateHint(other); got != other {
		t.Errorf("an error outside the state directory was changed: %q", got)
	}

	// Most commands return cli.Exit(err, …), whose error has no Unwrap: the
	// hint reads the message, and the exit code stays the command's.
	exited := stateHint(cli.Exit(err, 2))
	if !strings.Contains(exited.Error(), "XDG_STATE_HOME is not set") || exitCode(exited) != 2 {
		t.Errorf("through cli.Exit: %q, exit %d", exited, exitCode(exited))
	}
	if got := stateHint(cli.Exit(other, 2)); got.Error() != other.Error() {
		t.Errorf("an error outside the state directory was changed through cli.Exit: %q", got)
	}
}
