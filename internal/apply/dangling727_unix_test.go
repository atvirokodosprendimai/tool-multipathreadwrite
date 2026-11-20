//go:build !windows

package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #727 review P2-2. A dangling absolute link inside the root made the held-root
// load say "path escapes from parent", so an unlink or a rename of it ended at
// exit 2 where it had been refused on its hunk, exit 1, as a file that does not
// exist.
func TestAnUnlinkOrRenameOfADanglingAbsoluteLinkIsRefusedAsMissing(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "nowhere.txt"), filepath.Join(root, "dangabs")); err != nil {
		t.Fatal(err)
	}
	for name, in := range map[string]Input{
		"unlink": {Path: "dangabs", Op: "unlink", Lines: -1},
		"rename": {Path: "dangabs", Op: "rename", Body: []string{"moved"}, Lines: -1},
	} {
		res, err := Apply(root, []Input{in}, Options{Force: true})
		if err != nil {
			t.Errorf("%s: a refusal came back as an error: %v", name, err)
			continue
		}
		if len(res.Hunks) != 1 || res.Hunks[0].Status != StatusFailed || !strings.Contains(res.Hunks[0].Reason, "does not exist") || strings.Contains(res.Hunks[0].Reason, "escapes") {
			t.Errorf("%s: %+v, want a hunk refused as a file that does not exist", name, res.Hunks)
		}
	}
}
