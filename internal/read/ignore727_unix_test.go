//go:build !windows

package read

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
)

// #727. A directory's .gitignore was opened by name after its link check: a
// directory swapped for a link out of the checkout in between let rules from
// outside it decide what the walk served. The inside .gitignore ignores
// nothing; the outside one ignores every .txt.
func TestAGitignoreUnderADirectorySwappedForALinkOutIsNotRead(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{".git", "d"} {
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for p, body := range map[string]string{filepath.Join(root, "d", ".gitignore"): "nomatch\n", filepath.Join(outside, ".gitignore"): "*.txt\n"} {
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var done bool
	prev := afterIgnoreCheckFn
	afterIgnoreCheckFn = func() {
		afterIgnoreCheckFn = prev
		if done {
			return
		}
		done = true
		if err := os.Rename(filepath.Join(root, "d"), filepath.Join(root, "d.old")); err != nil {
			t.Error(err)
			return
		}
		if err := os.Symlink(outside, filepath.Join(root, "d")); err != nil {
			t.Error(err)
		}
	}
	t.Cleanup(func() { afterIgnoreCheckFn = prev })
	ig := newIgnorer(root)
	if ig == nil {
		t.Fatal("no ignorer")
	}
	if ig.Ignored("d/f.txt", false, 0) {
		t.Error("rules from a .gitignore outside the checkout ignored a file")
	}
}

// The pair: a .gitignore of the checkout still applies.
func TestAGitignoreInsideTheCheckoutStillApplies(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{".git", "d"} {
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "d", ".gitignore"), []byte("*.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ig := newIgnorer(root)
	if ig == nil || !ig.Ignored("d/f.txt", false, 0) {
		t.Error("a .gitignore inside the checkout did not apply")
	}
}

// #727. .git/info/exclude was read by name: an info directory that is a link out
// of the checkout let rules from outside it decide what the walk served.
func TestAnExcludeFileBehindALinkOutOfTheCheckoutIsNotRead(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "exclude"), []byte("*.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".git", "info")); err != nil {
		t.Fatal(err)
	}
	if ig := newIgnorer(root); ig == nil || ig.Ignored("a.txt", false, 0) {
		t.Error("rules from outside the checkout ignored a file")
	}
}

// The pair: .git/info/exclude inside the checkout still applies.
func TestAnExcludeFileInsideTheCheckoutStillApplies(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".git", "info"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "info", "exclude"), []byte("*.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ig := newIgnorer(root); ig == nil || !ig.Ignored("a.txt", false, 0) {
		t.Error(".git/info/exclude did not apply")
	}
}

// ADR-077, #727. With mrw's state directory inside the root, a file the walk
// discovered that is swapped for a link into the state is judged again, state
// check included, and refused. The state file holds the pattern; the inside
// file does not, so a spec for d/f.txt is the leak.
func TestAWalkedFileSwappedIntoMrwsStateIsRefused(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, ".st"))
	k := filepath.Join(root, ".st", "mrw", "k")
	for _, d := range []string{k, filepath.Join(root, "d")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(k, "f.txt"), []byte("needle\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "d", "f.txt"), []byte("nothing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var done bool
	prev := scanRead
	scanRead = func(rd *rooted.Reader, path string, budget int64) ([]byte, error) {
		if !done && filepath.Base(path) == "f.txt" && filepath.Base(filepath.Dir(path)) == "d" {
			done = true
			if err := os.Rename(filepath.Join(root, "d"), filepath.Join(root, "d.old")); err != nil {
				t.Error(err)
			}
			if err := os.Symlink(filepath.Join(".st", "mrw", "k"), filepath.Join(root, "d")); err != nil {
				t.Error(err)
			}
		}
		return prev(rd, path, budget)
	}
	t.Cleanup(func() { scanRead = prev })
	specs, _, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("needle")})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range specs {
		if strings.HasPrefix(s.Path, "d") {
			t.Errorf("a file of mrw's state was matched through a swapped directory: %+v", s)
		}
	}
}
