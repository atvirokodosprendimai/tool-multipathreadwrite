package bench

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAPrivateIndexSeesASameSizeEditGitWouldHaveRechecked holds #816. git takes
// an index entry whose size and mtime match the file as unchanged, unless the
// entry is no older than the index file itself ("racily clean"), when it reads
// the content. A copy of the tree's index made with a fresh mtime made a racily
// clean entry look settled, so a same-size edit made in the second the tree was
// checked out was dropped from the diff count and from the kept diff. The race
// is fixed here: ctime ignored and only whole-second mtime and size compared,
// the file and the index entry and the index file all at one second.
func TestAPrivateIndexSeesASameSizeEditGitWouldHaveRechecked(t *testing.T) {
	t.Setenv("GIT_CONFIG_COUNT", "2")
	t.Setenv("GIT_CONFIG_KEY_0", "core.trustctime")
	t.Setenv("GIT_CONFIG_VALUE_0", "false")
	t.Setenv("GIT_CONFIG_KEY_1", "core.checkStat")
	t.Setenv("GIT_CONFIG_VALUE_1", "minimal")
	tree, base := stagedTree(t)
	f := filepath.Join(tree, "f.txt")
	stamp := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(f, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	if err := git(tree, "update-index", "--refresh"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f, []byte("g\n"), 0o644); err != nil { // the same size as "f\n"
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "created.txt"), []byte("c\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{f, filepath.Join(tree, ".git", "index")} {
		if err := os.Chtimes(p, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	if out, err := gitOut(tree, "diff", "--name-only"); err != nil || strings.TrimSpace(out) != "f.txt" {
		t.Fatalf("control: git on the tree's own index should see the edit, got %q %v", out, err)
	}
	d, err := diffStat(t.TempDir(), tree, base)
	if err != nil || d.Files != 2 {
		t.Errorf("diffStat: %+v %v, want 2 files: f.txt edited to the same size, and created.txt", d, err)
	}
	keptBase := filepath.Join(t.TempDir(), "attempt")
	if _, note, err := keepDiff(t.TempDir(), tree, base, keptBase); err != nil || note != "" {
		t.Fatalf("keepDiff: %q %v", note, err)
	}
	kept, err := os.ReadFile(keptBase + ".diff")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"+++ b/f.txt", "+g", "+++ b/created.txt"} {
		if !strings.Contains(string(kept), want) {
			t.Errorf("the kept diff lacks %q:\n%s", want, kept)
		}
	}
}
