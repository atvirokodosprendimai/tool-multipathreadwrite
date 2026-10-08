//go:build windows

package rooted

import (
	"path/filepath"
	"testing"
)

// ADR-131 on Windows: a junction is followed by throughLinks, not by
// EvalSymlinks (ADR-071), so a Resolver must follow it in the directory it
// caches. Out of the root, inside it, and as the root itself, every path is
// answered as Resolve answers it, twice.
func TestAResolverFollowsAJunctionAsResolveDoes(t *testing.T) {
	base := t.TempDir()
	root, outside := filepath.Join(base, "root"), filepath.Join(base, "outside")
	mustWrite(t, filepath.Join(outside, "secret.txt"), "secret\n")
	mustWrite(t, filepath.Join(root, "inner", "ok.txt"), "ok\n")
	mklinkJ(t, filepath.Join(root, "j"), outside)
	mklinkJ(t, filepath.Join(root, "in"), filepath.Join(root, "inner"))
	alias := filepath.Join(base, "alias")
	mklinkJ(t, alias, root)

	for _, r := range []string{root, alias} {
		res := NewResolver(r)
		for round := 0; round < 2; round++ {
			for _, p := range []string{`j\secret.txt`, `j\new.txt`, `in\ok.txt`, `inner\ok.txt`, `j`, `in`} {
				want, wantErr := Resolve(r, p)
				got, gotErr := res.Resolve(p)
				if got != want || (gotErr == nil) != (wantErr == nil) || (gotErr != nil && gotErr.Error() != wantErr.Error()) {
					t.Errorf("root %s, round %d, %s: Resolver answered (%q, %v), Resolve (%q, %v)", r, round, p, got, gotErr, want, wantErr)
				}
			}
		}
	}
}
