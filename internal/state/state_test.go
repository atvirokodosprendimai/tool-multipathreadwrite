package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// xdg points XDG_STATE_HOME at a temp dir for the duration of one test.
func xdg(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	t.Setenv("XDG_STATE_HOME", base)
	return base
}

func TestDirIsUnderXDGStateHome(t *testing.T) {
	base := xdg(t)
	root := t.TempDir()

	dir, err := Dir(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(dir, filepath.Join(base, "mrw")) {
		t.Errorf("Dir = %q, want it under %q", dir, filepath.Join(base, "mrw"))
	}
	// The whole point: never under the tree it is about.
	if strings.HasPrefix(dir, root) {
		t.Errorf("Dir = %q is inside the root %q", dir, root)
	}
}

func TestDirFallsBackToLocalState(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir, err := Dir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".local", "state", "mrw"); !strings.HasPrefix(dir, want) {
		t.Errorf("Dir = %q, want it under %q", dir, want)
	}
}

// A relative XDG_STATE_HOME is refused rather than normalised: the spec says it
// must be absolute, and honouring a relative one would resolve against the
// working directory — which is how state lands back in the tree, the bug this
// package exists to fix.
func TestRelativeXDGStateHomeIsRejected(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "relative/path")
	if dir, err := Dir(t.TempDir()); err == nil {
		t.Errorf("Dir = %q, want an error for a relative XDG_STATE_HOME", dir)
	}
}

func TestDirIsStableAndPerCheckout(t *testing.T) {
	xdg(t)
	a, b := t.TempDir(), t.TempDir()

	first, err := Dir(a)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Dir(a)
	if err != nil {
		t.Fatal(err)
	}
	if first != again {
		t.Errorf("Dir is not stable: %q then %q", first, again)
	}
	other, err := Dir(b)
	if err != nil {
		t.Fatal(err)
	}
	if other == first {
		t.Error("two different roots share a state directory")
	}
}

func TestDirRecordsItsRoot(t *testing.T) {
	xdg(t)
	root := t.TempDir()

	dir, err := Dir(root)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "root"))
	if err != nil {
		t.Fatalf("no root marker: %v — an orphaned state dir would be an anonymous hash", err)
	}
	// EvalSymlinks because t.TempDir() is under /var -> /private/var on darwin.
	want, _ := filepath.EvalSymlinks(root)
	if got := strings.TrimSpace(string(b)); got != want {
		t.Errorf("root marker = %q, want %q", got, want)
	}
}

func TestMigrateCopiesLegacyStateWithoutDeleting(t *testing.T) {
	xdg(t)
	root := t.TempDir()
	legacy := filepath.Join(root, ".mrw")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "seen"), []byte("abc  a.go\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	moved, err := Migrate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(moved) != 1 || moved[0] != "seen" {
		t.Errorf("Migrate reported %v, want [seen]", moved)
	}
	dir, _ := Dir(root)
	b, err := os.ReadFile(filepath.Join(dir, "seen"))
	if err != nil || string(b) != "abc  a.go\n" {
		t.Errorf("legacy ledger not copied: %v %q", err, b)
	}
	// Never deletes: removing a file a caller may have committed is a worse bug
	// than the one being fixed.
	if _, err := os.Stat(filepath.Join(legacy, "seen")); err != nil {
		t.Errorf("Migrate deleted the legacy file: %v", err)
	}
}

func TestMigrateNeverOverwritesNewerState(t *testing.T) {
	xdg(t)
	root := t.TempDir()
	dir, err := Dir(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "seen"), []byte("current\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(root, ".mrw")
	_ = os.MkdirAll(legacy, 0o755)
	_ = os.WriteFile(filepath.Join(legacy, "seen"), []byte("stale\n"), 0o644)

	moved, err := Migrate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(moved) != 0 {
		t.Errorf("Migrate reported %v, want nothing — the destination already had state", moved)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "seen"))
	if string(b) != "current\n" {
		t.Errorf("live state was clobbered by a legacy file: %q", b)
	}
}

// The guarantee, as a property rather than a list of filenames: resolving and
// using state must leave the tree untouched. A list would need updating every
// time the tool learns to store something new; this does not.
func TestNoStateIsWrittenUnderTheRepoRoot(t *testing.T) {
	xdg(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dir, err := Dir(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "seen"), []byte("x  a.go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Migrate(root); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 1 || names[0] != "a.go" {
		t.Errorf("the root holds %v, want only [a.go] — mrw left something behind", names)
	}
}
