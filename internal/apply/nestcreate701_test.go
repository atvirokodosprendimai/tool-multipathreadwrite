package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTwoWrittenPathsWhereOneSitsUnderTheOtherAreRefused: creates of ok.txt,
// new and new/child.txt passed validation and a dry run, then staging made
// directory new, ok.txt landed and the commit failed: PARTIALLY APPLIED, exit
// 2 (#701). A plan cannot make one path both a file and a directory, so it is
// refused before anything is staged, a dry run included. The pair: the child
// alone, and the file alone, apply.
func TestTwoWrittenPathsWhereOneSitsUnderTheOtherAreRefused(t *testing.T) {
	create := func(p string) Input {
		return Input{Path: filepath.FromSlash(p), Op: "create", Body: []string{"x"}, Lines: -1}
	}
	for _, dry := range []bool{true, false} {
		root := t.TempDir()
		res, err := Apply(root, []Input{create("ok.txt"), create("new"), create("new/child.txt")}, Options{DryRun: dry})
		if err != nil {
			t.Fatalf("dry run %v: %v", dry, err)
		}
		if res.Applied || res.Failed == 0 {
			t.Fatalf("dry run %v: a plan writing new and new/child.txt validated: %+v", dry, res.Hunks)
		}
		var named bool
		for _, h := range res.Hunks {
			if h.Status == StatusFailed && strings.Contains(h.Reason, "sits under") {
				named = true
			}
			if h.Path == "ok.txt" && h.Status != StatusSkipped {
				t.Errorf("dry run %v: the sibling ok.txt reports %s, want skip", dry, h.Status)
			}
		}
		if !named {
			t.Errorf("dry run %v: no hunk names the nesting: %+v", dry, res.Hunks)
		}
		if ents, _ := os.ReadDir(root); len(ents) != 0 {
			t.Errorf("dry run %v: the refused plan left %d entries in the root", dry, len(ents))
		}
	}
	for _, p := range []string{"new", "new/child.txt"} {
		root := t.TempDir()
		res, err := Apply(root, []Input{create("ok.txt"), create(p)}, Options{})
		if err != nil || !res.Applied {
			t.Errorf("the pair: ok.txt and %s alone did not apply: %v %+v", p, err, res.Hunks)
		}
	}
}
