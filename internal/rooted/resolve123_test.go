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

// The review of #330: the cache checked the old real path, so a symlink above
// the base re-pointed during a session left the old directory standing and the
// new base was served. The entry stands only while the base, followed now, is
// the directory it was made for.
func TestAStateBaseBehindARepointedLinkIsRefused(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"s1/mrw/k", "s2/mrw/k"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(root, "st")
	if err := os.Symlink(filepath.Join(root, "s1"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	t.Setenv("XDG_STATE_HOME", link)
	if _, err := Resolve(root, "s1/mrw/k"); err == nil || !strings.Contains(err.Error(), "own state") {
		t.Fatalf("the base behind the link was served: %v", err)
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "s2"), link); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(root, "s2/mrw/k"); err == nil || !strings.Contains(err.Error(), "own state") {
		t.Fatalf("after the link was re-pointed, the new base was served: %v", err)
	}
}
