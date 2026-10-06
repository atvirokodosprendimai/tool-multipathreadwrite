package read

import (
	"testing"
)

// The Codex review of v1.42.0..v1.47.0, finding 3. Under --no-ignore the
// ast-grep filter was turned off whole, so a hit inside .git was served; the
// walk prunes a .git it meets whatever --no-ignore says, and a discovered hit
// there is dropped now too. What --no-ignore turns off is the ignore rules and
// the binary skip, not .git.
func TestAstGrepUnderNoIgnoreStillDropsAHitInsideGit(t *testing.T) {
	root, hits := tree122(t)
	installRecordingAstGrep(t, hits)
	specs, _, err := AstGrep(root, nil, "package $A", nil, AstGrepOptions{NoIgnore: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range servedPaths(specs) {
		if p == ".git/objects/obj.go" {
			t.Fatalf("under NoIgnore a hit inside .git was served: %v", servedPaths(specs))
		}
	}
	// A .git the caller names is walked, as --grep walks it: only a .git below
	// the named start is dropped (both reviews of #337).
	specs, _, err = AstGrep(root, []string{".git"}, "package $A", nil, AstGrepOptions{NoIgnore: true})
	if err != nil {
		t.Fatal(err)
	}
	// The fake answers every hit whatever it is given, so only the .git hit is
	// asserted: it is served under the named .git.
	served := false
	for _, p := range servedPaths(specs) {
		served = served || p == ".git/objects/obj.go"
	}
	if !served {
		t.Errorf("under a named .git the hit inside it was not served: %v", servedPaths(specs))
	}
}
