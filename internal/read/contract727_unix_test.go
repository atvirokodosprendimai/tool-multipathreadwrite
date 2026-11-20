//go:build !windows

package read

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func realTemp727(t *testing.T) string {
	t.Helper()
	p, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// #727 review (contract). A root that is not there is refused on its row, as it
// always was; the walk holding a root must not turn it into an error that
// costs the caller the row and the exit code.
func TestAWalkOfAMissingRootIsRefusedOnItsRowNotAnError(t *testing.T) {
	root := filepath.Join(realTemp727(t), "gone")
	_, problems, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("x")})
	if err != nil {
		t.Fatalf("the walk of a missing root returned an error: %v", err)
	}
	if len(problems) == 0 || !strings.Contains(problems[0].Reason, "does not exist") {
		t.Errorf("problems %+v, want the root named as missing", problems)
	}
}

// A named miss keeps the whole lookup path in its words, as the open by name gave.
func TestANamedMissBelowAFileNamesTheFullLookupPath(t *testing.T) {
	root := realTemp727(t)
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	Run(&out, root, []Spec{{Path: "README.md/nope"}}, Options{})
	want := "open " + filepath.Join(root, "README.md", "nope") + ": not a directory"
	if !strings.Contains(out.String(), want) {
		t.Errorf("output %q does not say %q", out.String(), want)
	}
}

// A dangling absolute link to a missing directory of the root, met in the
// middle of the path, is a file that does not exist, not a path that escapes.
func TestADanglingAbsoluteDirectoryLinkInTheMiddleIsAMissingFile(t *testing.T) {
	root := realTemp727(t)
	if err := os.Symlink(filepath.Join(root, "missingdir"), filepath.Join(root, "dabs")); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	Run(&out, root, []Spec{{Path: "dabs/f.txt"}}, Options{})
	want := "open " + filepath.Join(root, "dabs", "f.txt") + ": no such file or directory"
	if !strings.Contains(out.String(), want) || strings.Contains(out.String(), "escapes") {
		t.Errorf("output %q does not say %q", out.String(), want)
	}
}

// #727 review (ignore rules). The exclude file behind an absolute link inside
// the checkout, and one below a directory the process may search but not read,
// apply as they did when it was opened by name.
func TestAnExcludeFileBehindAnAbsoluteInRootLinkOrASearchOnlyDirectoryStillApplies(t *testing.T) {
	root := realTemp727(t)
	if err := os.MkdirAll(filepath.Join(root, ".git", "real"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".git", "info"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "real", "x"), []byte("*.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, ".git", "real", "x"), filepath.Join(root, ".git", "info", "exclude")); err != nil {
		t.Fatal(err)
	}
	if ig := newIgnorer(root); ig == nil || !ig.Ignored("a.txt", false, 0) {
		t.Error("an exclude file behind an absolute link inside the checkout did not apply")
	}
	if os.Geteuid() == 0 {
		return
	}
	root2 := realTemp727(t)
	if err := os.MkdirAll(filepath.Join(root2, ".git", "info"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root2, ".git", "info", "exclude"), []byte("*.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(root2, ".git", "info"), 0o111); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(root2, ".git", "info"), 0o755) })
	if ig := newIgnorer(root2); ig == nil || !ig.Ignored("a.txt", false, 0) {
		t.Error("an exclude file in a search-only directory did not apply")
	}
}

// #727 review (ast-grep preflight). A named directory swapped for a link out of
// the root after it was judged is not probed: the probe opens through the held
// root, the path is refused, and ast-grep is never started.
func TestTheAstGrepPreflightDoesNotProbeANamedDirectorySwappedForALinkOut(t *testing.T) {
	root, outside := realTemp727(t), realTemp727(t)
	if err := os.Mkdir(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "f.go"), []byte("package f\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	argv := installRecordingAstGrep(t, "[]")
	var done bool
	prev := afterJudgeNamedFn
	afterJudgeNamedFn = func() {
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
	t.Cleanup(func() { afterJudgeNamedFn = prev })
	if _, _, err := AstGrep(root, []string{"d"}, "x", nil, AstGrepOptions{NoIgnore: true}); err != nil {
		t.Fatal(err)
	}
	if ops, ran := astGrepOperands(t, argv); ran {
		t.Errorf("ast-grep was started with %v for a directory swapped out of the root", ops)
	}
}
