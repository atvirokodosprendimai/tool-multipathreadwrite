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
}
