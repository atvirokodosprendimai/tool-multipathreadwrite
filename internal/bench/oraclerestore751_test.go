package bench

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// coupled751 builds a check-coupled task whose package is pkg ("p", or "." for
// the repository root) and returns a grader of a change to its tree.
func coupled751(t *testing.T, pkg string) (grade func(change map[string]string) Verdict, fixed map[string]string, name func(string) string) {
	t.Helper()
	needGit(t)
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go is not on PATH")
	}
	name = func(f string) string { return path.Join(pkg, f) }
	repo := t.TempDir()
	write := func(root string, fs map[string]string) {
		t.Helper()
		for n, body := range fs {
			p := filepath.Join(root, filepath.FromSlash(n))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	git := func(dir string, args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=b", "-c", "user.email=b@b"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	src := "package p\n\n// Status is the state a new subscription takes.\nfunc Status() string { return \"active\" }\n"
	tst := "package p\n\nimport \"testing\"\n\nfunc TestStatus(t *testing.T) {\n\tif Status() != \"active\" {\n\t\tt.Fatalf(\"Status() = %q\", Status())\n\t}\n}\n"
	write(repo, map[string]string{"go.mod": "module example.com/m\n\ngo 1.21\n", name("p.go"): src, name("p_test.go"): tst})
	git(repo, "init", "-q")
	git(repo, "add", "-A")
	git(repo, "commit", "-qm", "base")
	sha := git(repo, "rev-parse", "HEAD")
	fixed = map[string]string{name("p.go"): strings.Replace(src, `"active"`, `"Active"`, 1), name("p_test.go"): strings.Replace(tst, `Status() != "active"`, `Status() != "Active"`, 1)}
	write(repo, fixed)
	patch := git(repo, "diff") + "\n"
	git(repo, "checkout", "-q", "--", ".")
	taskDir := filepath.Join(t.TempDir(), "am-coupled-p-active")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for n, body := range map[string]string{
		"reference.patch": patch,
		"task.json": `{"id": "am-coupled-p-active", "shape": "check-coupled", "repo": "m", "commit": "` + sha + `",
  "instruction": "In ` + name("p.go") + `, change the string \"active\" to \"Active\", and keep the tests passing.",
  "test_package": "` + pkg + `", "asserting_tests": ["TestStatus"],
  "require": [{"file": "` + name("p.go") + `", "match": "\"Active\""}], "forbid": [{"file": "` + name("p.go") + `", "match": "\"active\""}]}`,
	} {
		if err := os.WriteFile(filepath.Join(taskDir, n), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	task, err := LoadTask(taskDir)
	if err != nil {
		t.Fatal(err)
	}
	grade = func(change map[string]string) Verdict {
		t.Helper()
		tree := t.TempDir()
		git(tree, "clone", "-q", repo, ".")
		write(tree, change)
		v, err := Oracle(task, tree, "HEAD")
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	return grade, fixed, name
}

// Review of #751. The tests that grade come back before the structural check: a
// test file the agent left that does not parse is replaced or removed, so it
// cannot fail the grade of a correct fix, and a fix that does not parse still does.
func TestABrokenTestFileTheAgentLeftDoesNotFailACorrectFix(t *testing.T) {
	grade, fixed, name := coupled751(t, "p")
	broken := map[string]string{name("p.go"): fixed[name("p.go")], name("p_test.go"): "package p\n\nfunc (", name("zz_test.go"): "package p\n\nfunc ("}
	if v := grade(broken); !v.Solved {
		t.Errorf("a broken test file the agent left fails a correct fix: %+v", v)
	}
	// The pair: the same breakage in the source is still refused.
	if v := grade(map[string]string{name("p.go"): "package p\n\nfunc ("}); v.Solved || len(v.Structure) == 0 {
		t.Errorf("a source file that does not parse is not refused: %+v", v)
	}
}

// A task whose package is the repository root keys its files without a directory:
// the restored base test is kept, not removed as one the agent added.
func TestATaskWhosePackageIsTheRepositoryRootKeepsItsRestoredTests(t *testing.T) {
	grade, fixed, name := coupled751(t, ".")
	if v := grade(fixed); !v.Solved {
		t.Errorf("the reference is unsolved for a root package: %+v", v)
	}
	if v := grade(map[string]string{name("p.go"): fixed[name("p.go")]}); !v.Solved {
		t.Errorf("a source-only fix is unsolved for a root package: %+v", v)
	}
	cheat := map[string]string{name("p.go"): "package p\n\nfunc Status() string { return \"Zctive\" }\n", name("p_test.go"): "package p\n\nimport \"testing\"\n\nfunc TestStatus(t *testing.T) {}\n"}
	if v := grade(cheat); v.Solved {
		t.Errorf("a weakened asserting test solves a root-package task: %+v", v)
	}
}
