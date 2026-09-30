package state

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
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
// and no temp stays beside it. Nothing writes the name in place: a rename
// still refused after its tries fails with the old file untouched (the review
// of the record found the first draft's in-place fallback reopened the torn
// write), a reader holding the file open only delays the rename, and a
// read-only file is refused.
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
	// On Windows os.SameFile reads a file's ID lazily, by path, at comparison
	// time — after the rename both FileInfos would load the new file's ID and
	// compare equal. Comparing before with itself loads its ID now (CI, #297).
	_ = os.SameFile(before, before)
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
	tries := 0
	renameFn = func(string, string) error { tries++; return errors.New("rename refused") }
	if err := Write(name, []byte("third\n"), 0o600); err == nil {
		t.Error("a rename refused every time was reported as a write")
	}
	if tries != renameTries {
		t.Errorf("the rename was tried %d time(s), want %d", tries, renameTries)
	}
	if b, _ := os.ReadFile(name); string(b) != "new ledger\n" {
		t.Errorf("after a refused rename the file holds %q, want the old bytes whole", b)
	}
	if left := tempsIn(t, dir); len(left) != 0 {
		t.Errorf("a refused rename left temp files: %q", left)
	}

	// A reader holding the file open — which on Windows refuses the rename —
	// only delays the write, because the rename is tried again.
	renameFn = real
	r, err := os.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(3 * renameWait)
		_ = r.Close()
	}()
	if err := Write(name, []byte("fourth\n"), 0o600); err != nil {
		t.Errorf("a write behind a reader that let go was refused: %v", err)
	}
	if b, _ := os.ReadFile(name); string(b) != "fourth\n" {
		t.Errorf("behind a reader the file holds %q, want the new bytes", b)
	}

	// A state file its owner made read-only is refused and unchanged, as it
	// was when it was rewritten in place.
	if err := os.Chmod(name, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(name, 0o600) })
	if err := Write(name, []byte("fifth\n"), 0o600); err == nil {
		t.Error("a read-only state file was replaced")
	}
	if b, _ := os.ReadFile(name); string(b) != "fourth\n" {
		t.Errorf("the read-only file holds %q, want it unchanged", b)
	}
}

// ADR-105 T2. Every state write goes through Write: no non-test source in the
// packages that keep state calls os.WriteFile, Write included. A new state
// file written in place would reopen the torn-read window unnoticed, so the
// class is checked, not the eight sites one by one.
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
			if n != 0 {
				t.Errorf("%s calls os.WriteFile %d time(s) outside state.Write", f, n)
			}
		}
	}
	if scanned < 10 {
		t.Fatalf("scanned %d source files; the scan did not reach the packages it guards", scanned)
	}
}
