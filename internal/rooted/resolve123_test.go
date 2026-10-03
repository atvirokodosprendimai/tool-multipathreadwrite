package rooted

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-123. Resolve caches the state base it resolved. A base removed and made
// again — another directory at the same path — must still be recognised by
// identity, on a filesystem that folds case, where only identity can tell
// that .st/MRW is the base.
func TestACaseSpellingOfARecreatedStateBaseIsRefused(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, ".st"))
	base := filepath.Join(root, ".st", "mrw")
	// Resolved once while the base is absent, so a cache entry made then must
	// not stand for the base once it exists.
	if _, err := Resolve(root, "x.txt"); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, "k"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".st", "MRW")); err != nil {
		t.Skip("the filesystem keeps case: .st/MRW is another directory")
	}
	if _, err := Resolve(root, ".st/MRW/k"); err == nil || !strings.Contains(err.Error(), "own state") {
		t.Fatalf("a case spelling of the state base was served: %v", err)
	}
	if err := os.RemoveAll(base); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, "k"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(root, ".st/MRW/k"); err == nil || !strings.Contains(err.Error(), "own state") {
		t.Fatalf("after the base was made again, a case spelling of it was served: %v", err)
	}
}
