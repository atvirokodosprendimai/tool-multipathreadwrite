package state

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// tempsIn names the files in dir that look like Write's temp files.
func tempsIn(t *testing.T, dir string) []string {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range ents {
		if strings.Contains(e.Name(), ".tmp-") {
			out = append(out, e.Name())
		}
	}
	return out
}

// ADR-105 T2. A state file was truncated and rewritten in place, so a reader
// that ran in between — or a run killed in the middle — saw part of it; for
// the ledger that is a lost licence. Write replaces the file by rename: the
// name then holds the new bytes as a different file, with the mode asked for,
// and no temp stays beside it. A rename the platform refuses (Windows, over a
// file another process holds open) falls back to the in-place write and still
// leaves no temp.
func TestAStateFileIsReplacedWholeNeverRewrittenInPlace(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "seen")
	if err := os.WriteFile(name, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(name, []byte("new ledger\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(name); string(b) != "new ledger\n" {
		t.Errorf("the file holds %q, want the new bytes", b)
	}
	if os.SameFile(before, after) {
		t.Error("the file was rewritten in place: it is the same file as before the write")
	}
	if runtime.GOOS != "windows" && after.Mode().Perm() != 0o600 {
		t.Errorf("mode %v, want 0600", after.Mode().Perm())
	}
	if left := tempsIn(t, dir); len(left) != 0 {
		t.Errorf("temp files left beside the state file: %q", left)
	}

	real := renameFn
	t.Cleanup(func() { renameFn = real })
	renameFn = func(string, string) error { return errors.New("rename refused") }
	if err := Write(name, []byte("third\n"), 0o600); err != nil {
		t.Fatalf("a refused rename did not fall back: %v", err)
	}
	if b, _ := os.ReadFile(name); string(b) != "third\n" {
		t.Errorf("after the fallback the file holds %q, want the new bytes", b)
	}
	if left := tempsIn(t, dir); len(left) != 0 {
		t.Errorf("the fallback left temp files: %q", left)
	}

	// A state file its owner made read-only stays refused and unchanged, as
	// it was when it was rewritten in place. uid 0 writes through the mode.
	renameFn = real
	if err := os.Chmod(name, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(name, 0o600) })
	if err := Write(name, []byte("fourth\n"), 0o600); os.Geteuid() != 0 && err == nil {
		t.Error("a read-only state file was replaced")
	}
	if b, _ := os.ReadFile(name); os.Geteuid() != 0 && string(b) != "third\n" {
		t.Errorf("the read-only file holds %q, want it unchanged", b)
	}
}

// ADR-105 T2. Every state write goes through Write: no non-test source in the
// packages that keep state calls os.WriteFile, except Write's own fallback.
// A new state file written in place would reopen the torn-read window
// unnoticed, so the class is checked, not the eight sites one by one.
func TestNoStateWriteBypassesTheAtomicWriter(t *testing.T) {
	scanned := 0
	for _, dir := range []string{"../seen", "../authoring", "../iter", "../mcp", "."} {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			scanned++
			n := strings.Count(string(b), "os.WriteFile(")
			if dir == "." && filepath.Base(f) == "write.go" {
				n-- // the fallback, when a rename is refused
			}
			if n != 0 {
				t.Errorf("%s calls os.WriteFile %d time(s) outside state.Write", f, n)
			}
		}
	}
	if scanned < 10 {
		t.Fatalf("scanned %d source files; the scan did not reach the packages it guards", scanned)
	}
}
