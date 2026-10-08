package rooted

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// ADR-131. A Resolver answers every path exactly as Resolve does, through
// every hazard Resolve refuses: a link out of the root, a link into mrw's
// state, the state base inside the root and a case spelling of it, a directory
// link, a missing leaf, a name spelled as a directory. Each path is asked
// twice, so the second answer comes from the cache.
func TestAResolverAnswersAsResolveDoes(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, ".st"))
	for _, name := range []string{"a/b/c/f.txt", "a/b/g.txt", ".st/mrw/k/p.json", ".st/notes.txt", "top.txt"} {
		mustCreate(t, filepath.Join(root, filepath.FromSlash(name)))
	}
	mustCreate(t, filepath.Join(outside, "secret.txt"))
	paths := []string{
		"a/b/c/f.txt", "a/b/g.txt", "top.txt", ".st/notes.txt", ".st/mrw/k/p.json", ".st/MRW/k/p.json",
		".st/mrw", "a/b/c", "a/b/c/missing.txt", "a/nodir/x.txt", "a/b/c/f.txt/", "a/b/", "../x.txt",
	}
	links := map[string]string{
		"a/in.txt":      filepath.Join(root, "a", "b", "c", "f.txt"),
		"a/out.txt":     filepath.Join(outside, "secret.txt"),
		"a/state.txt":   filepath.Join(root, ".st", "mrw", "k", "p.json"),
		"a/dirout":      outside,
		"a/dirin":       filepath.Join(root, "a", "b"),
		"a/b/c/rel.txt": "f.txt",
	}
	linked := true
	for name, target := range links {
		if err := os.Symlink(target, filepath.Join(root, filepath.FromSlash(name))); err != nil {
			linked = false
			t.Logf("symlinks unavailable, the link rows are dropped: %v", err)
			break
		}
	}
	if linked {
		paths = append(paths, "a/in.txt", "a/out.txt", "a/state.txt", "a/dirout", "a/dirout/secret.txt",
			"a/dirout/new.txt", "a/dirin/c/f.txt", "a/b/c/rel.txt")
	}

	r := NewResolver(root)
	for round := 0; round < 2; round++ {
		for _, p := range paths {
			p = filepath.FromSlash(p)
			want, wantErr := Resolve(root, p)
			got, gotErr := r.Resolve(p)
			if got != want || (gotErr == nil) != (wantErr == nil) ||
				(gotErr != nil && gotErr.Error() != wantErr.Error()) ||
				errors.Is(gotErr, ErrDeviceName) != errors.Is(wantErr, ErrDeviceName) {
				t.Errorf("round %d, %s: Resolver answered (%q, %v), Resolve (%q, %v)", round, p, got, gotErr, want, wantErr)
			}
		}
	}

	// A root that cannot be resolved is refused for every path, as Resolve
	// refuses it.
	gone := filepath.Join(root, "no-such-root")
	_, wantErr := Resolve(gone, "x.txt")
	if _, err := NewResolver(gone).Resolve("x.txt"); err == nil || wantErr == nil || err.Error() != wantErr.Error() {
		t.Errorf("a missing root: Resolver said %v, Resolve %v", err, wantErr)
	}
}

func mustCreate(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
