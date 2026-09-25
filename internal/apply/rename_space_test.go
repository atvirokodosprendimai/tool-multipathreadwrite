package apply

import (
	"os"
	"path/filepath"
	"testing"
)

// ADR-069 T3. A rename's one-line body is its destination, and it was
// TrimSpaced at three sites, so a rename to "d " landed at d: a name the
// caller did not write.
func TestARenameDestinationKeepsItsTrailingSpace(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "src\n")
	if err := os.WriteFile(filepath.Join(root, "probe "), nil, 0o644); err != nil {
		t.Skipf("this filesystem cannot hold a trailing-space name: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "probe")); err == nil {
		t.Skip("this filesystem folds a trailing space")
	}
	res, err := Apply(root, []Input{
		{Path: "a.txt", Op: "rename", Body: []string{"d "}, Lines: -1},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Applied {
		t.Fatalf("the rename did not apply: %+v", res.Hunks)
	}
	if exists(t, root, "d") {
		t.Error("the rename landed at d, not at the destination the plan named")
	}
	if !exists(t, root, "d ") || read(t, root, "d ") != "src\n" {
		t.Error("the rename did not land at \"d \"")
	}
}
