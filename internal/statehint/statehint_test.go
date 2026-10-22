package statehint

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// hinted fails the test unless Hint(err) keeps err's text and carries each of
// want, with the name of the directory and the variable that moves it.
func hinted(t *testing.T, name string, err error, want ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: the fixture did not fail", name)
	}
	got := Hint(err)
	if !errors.Is(got, err) {
		t.Errorf("%s: the hint does not wrap the error it was given", name)
	}
	for _, w := range append([]string{err.Error(), "mrw's state directory", "read ledger", "set XDG_STATE_HOME to a writable absolute directory"}, want...) {
		if !strings.Contains(got.Error(), w) {
			t.Errorf("%s: %q lacks %q", name, got, w)
		}
	}
}

// A failure in mrw's state directory is named as mrw's, with the variable that
// moves it, whether XDG_STATE_HOME is set or not and whatever the call wrapped
// it in: Hint judges the error's type (*fs.PathError, *os.LinkError), never its
// text. A file where a directory must go fails for any user, root included. The
// pair: an error that is not about the state directory is returned as it came,
// a refusal that echoes caller text naming the base among them.
func TestAStateFailureIsNamedByItsType(t *testing.T) {
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
	hinted(t, "wrapped with %w", fmt.Errorf("ack: %w", err), "under XDG_STATE_HOME="+base)

	// What seen.save returns when the ledger's rename fails: its words, then %w.
	ledger := filepath.Join(base, "mrw", "0123456789abcdef", "seen")
	renamed := &os.LinkError{Op: "rename", Old: ledger + ".tmp", New: ledger, Err: os.ErrPermission}
	hinted(t, "a ledger that could not be updated", fmt.Errorf("the read ledger could not be updated; whatever this call already did stands: %w", renamed))

	// What prune returns when the home itself cannot be opened: os.OpenRoot of
	// base's parent, here a file.
	file := filepath.Join(t.TempDir(), "home")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", file)
	_, err = os.OpenRoot(file)
	hinted(t, "prune opening the state home", err, file+" is a file, not a directory")

	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".local"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	_, err = state.Dir(t.TempDir())
	hinted(t, "a parent of the default base", err, "XDG_STATE_HOME is not set")

	b, err := state.Base()
	if err != nil {
		t.Fatal(err)
	}
	_, missing := os.ReadFile(filepath.Join(t.TempDir(), "missing"))
	if !errors.Is(missing, os.ErrNotExist) {
		t.Fatalf("the pair's fixture: %v", missing)
	}
	echoed := fmt.Errorf("unknown --format %q", "bad: open "+filepath.Join(b, "x"))
	for _, e := range []error{missing, echoed} {
		if got := Hint(e); got != e {
			t.Errorf("an error that is not about the state directory was changed: %q", got)
		}
	}
}

// A regular file where a directory must go made Windows say only "The system
// cannot find the path specified" (the v1.71.0 field test, two Windows peers):
// the hint now names the file, on every OS, so the caller knows what to move.
// A regular file as a path component fails mkdir on every OS, so the test needs
// no platform. The pairs: an error with no file in the way, and an existing
// file that merely could not be opened, say nothing about a file.
func TestAFileInTheWayOfTheStateDirectoryIsNamed(t *testing.T) {
	file := filepath.Join(t.TempDir(), "home")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", filepath.Join(file, "x"))
	_, err := state.Dir(t.TempDir())
	hinted(t, "a file as the state home's parent", err, file+" is a file, not a directory")

	// The ledger's own file could not be opened: it IS a file, and that is no news.
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	base, err := state.Base()
	if err != nil {
		t.Fatal(err)
	}
	ledger := filepath.Join(base, "mrw", "0123456789abcdef", "seen")
	if err := os.MkdirAll(filepath.Dir(ledger), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ledger, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for name, e := range map[string]error{
		"no file in the way":              &os.PathError{Op: "mkdir", Path: filepath.Join(base, "mrw", "nothing"), Err: os.ErrPermission},
		"a file that could not be opened": &os.PathError{Op: "open", Path: ledger, Err: os.ErrPermission},
	} {
		if got := Hint(e); !errors.Is(got, e) || strings.Contains(got.Error(), "is a file") {
			t.Errorf("%s: %q", name, got)
		}
	}
}
