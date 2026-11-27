package bench

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
// is fixed here: ctime ignored and only whole-second mtime and size compared (set
// in the repository's own config, which every git reads), the file and the index
// entry and the index file all at one second. A negative control shows the race
// is real: the same index bytes with a later mtime miss the edit.
func TestAPrivateIndexSeesASameSizeEditGitWouldHaveRechecked(t *testing.T) {
	tree, base := stagedTree(t)
	for _, kv := range [][2]string{{"core.trustctime", "false"}, {"core.checkStat", "minimal"}} {
		if err := git(tree, "config", "--file", filepath.Join(tree, ".git", "config"), kv[0], kv[1]); err != nil { // --file: a GIT_CONFIG in the environment must not take the write
			t.Fatal(err)
		}
	}
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
	newer := filepath.Join(t.TempDir(), "newer.index")
	idx, err := os.ReadFile(filepath.Join(tree, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newer, idx, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newer, stamp.Add(5*time.Second), stamp.Add(5*time.Second)); err != nil {
		t.Fatal(err)
	}
	c := exec.Command("git", "-C", tree, "diff", "--name-only")
	c.Env = append(os.Environ(), "GIT_INDEX_FILE="+newer)
	if out, err := c.Output(); err != nil || strings.TrimSpace(string(out)) != "" {
		t.Fatalf("negative control: the same index with a later mtime should trust the stat and miss the edit, got %q %v; the fixture does not reach the race", out, err)
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
	for _, want := range []string{`(?m)^\+\+\+ (b/)?f\.txt$`, `(?m)^\+g$`, `(?m)^\+\+\+ (b/)?created\.txt$`} {
		if !regexp.MustCompile(want).Match(kept) {
			t.Errorf("the kept diff lacks /%s/:\n%s", want, kept)
		}
	}
}
