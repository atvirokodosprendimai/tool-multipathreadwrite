package read

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// ADR-116 T1. A rule table for the native matcher, one row per gitignore(5)
// clause. Each row's file sits at base ("" is the checkout's top).
func TestIgnoreRules(t *testing.T) {
	type probe struct {
		path  string
		dir   bool
		ignor bool
	}
	for _, c := range []struct {
		name, base, rules string
		probes            []probe
	}{
		{"a glob matches at any depth", "", "*.log\n", []probe{{"a.log", false, true}, {"d/x.log", false, true}, {"a.txt", false, false}}},
		{"comments and escapes", "", "#c\n\\#h\n\\!x\n", []probe{{"#h", false, true}, {"#c", false, false}, {"!x", false, true}}},
		{"trailing spaces trimmed unless escaped", "", "a.txt   \nb\\ \n", []probe{{"a.txt", false, true}, {"b ", false, true}, {"b", false, false}}},
		{"a trailing slash means directories only", "", "build/\n", []probe{{"build", true, true}, {"build", false, false}, {"build/x", false, true}, {"d/build", true, true}}},
		{"a leading slash anchors", "", "/top.txt\n", []probe{{"top.txt", false, true}, {"d/top.txt", false, false}}},
		{"a middle slash anchors", "", "d/x.go\n", []probe{{"d/x.go", false, true}, {"e/d/x.go", false, false}}},
		{"negation re-includes", "", "*.go\n!keep.go\n", []probe{{"keep.go", false, false}, {"a.go", false, true}}},
		{"nothing under an ignored directory is re-included", "", "logs/\n!logs/keep.log\n", []probe{{"logs/keep.log", false, true}}},
		{"leading **", "", "**/foo\n", []probe{{"foo", false, true}, {"a/b/foo", false, true}}},
		{"trailing **", "", "a/**\n", []probe{{"a/x", false, true}, {"a/x/y", false, true}, {"a", true, false}}},
		{"middle **", "", "a/**/b\n", []probe{{"a/b", false, true}, {"a/x/y/b", false, true}, {"c/a/b", false, false}}},
		{"? and classes", "", "f?.txt\n[ab].c\n[!xy].d\n", []probe{{"f1.txt", false, true}, {"f12.txt", false, false}, {"a.c", false, true}, {"c.c", false, false}, {"z.d", false, true}, {"x.d", false, false}}},
		{"a nested file applies under its directory", "sub", "*.tmp\n/y\n", []probe{{"sub/x.tmp", false, true}, {"x.tmp", false, false}, {"sub/y", false, true}, {"sub/z/y", false, false}}},
		{"the last match wins", "", "a.txt\n!a.txt\na.txt\n", []probe{{"a.txt", false, true}}},
		{"** alone matches every path", "", "a*/**\n!/**\n", []probe{{"ab/a.x", false, false}, {"ab/c/d", false, false}}},
	} {
		ig := staticIgnorer(parseIgnore(c.base, []byte(c.rules), false))
		for _, p := range c.probes {
			if got := ig.Ignored(p.path, p.dir, 0); got != p.ignor {
				t.Errorf("%s: %q (dir %v) ignored %v, want %v", c.name, p.path, p.dir, got, p.ignor)
			}
		}
	}
}

// ignoreTree builds a tree for the walk tests: gitDir says whether it is a
// checkout.
func ignoreTree(t *testing.T, gitDir bool) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		".gitignore":        "node_modules/\n*.log\n",
		"node_modules/x.js": "needle\n",
		"a.log":             "needle\n",
		"b.go":              "needle\n",
		"bin.dat":           "nee\x00dle needle\n",
		"sub/.gitignore":    "secret.go\n",
		"sub/secret.go":     "needle\n",
		"sub/ok.go":         "needle\n",
		"excluded.go":       "needle\n",
	}
	if gitDir {
		files[".git/info/exclude"] = "excluded.go\n"
	}
	for n, b := range files {
		p := filepath.Join(root, n)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(b), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func specPaths(specs []Spec) []string {
	var out []string
	for _, s := range specs {
		out = append(out, s.Path)
	}
	sort.Strings(out)
	return out
}

// ADR-116 T1. A walk served every regular file, so --grep read node_modules,
// build output and binaries. Inside a checkout it skips what .gitignore and
// .git/info/exclude ignore, prunes an ignored directory, skips binaries, and
// counts each; a named path is served; NoIgnore walks as ADR-007 did; outside
// a checkout .gitignore is not applied, as git does not apply it.
func TestAWalkSkipsWhatGitIgnores(t *testing.T) {
	needle := regexp.MustCompile("needle")
	t.Run("inside a checkout", func(t *testing.T) {
		root := ignoreTree(t, true)
		var sk WalkSkipped
		specs, probs, err := Walk(root, nil, WalkOptions{Pattern: needle, Skipped: &sk})
		if err != nil || len(probs) != 0 {
			t.Fatalf("err %v problems %v", err, probs)
		}
		if got := strings.Join(specPaths(specs), " "); got != "b.go sub/ok.go" {
			t.Errorf("served %q, want b.go sub/ok.go", got)
		}
		if sk.IgnoredDirs != 1 || sk.Ignored != 3 || sk.Binary != 1 {
			t.Errorf("skipped %+v, want 1 directory, 3 files (a.log, sub/secret.go, excluded.go), 1 binary", sk)
		}
	})
	t.Run("a named ignored file is served", func(t *testing.T) {
		root := ignoreTree(t, true)
		specs, _, err := Walk(root, []string{"a.log"}, WalkOptions{Pattern: needle})
		if err != nil || strings.Join(specPaths(specs), " ") != "a.log" {
			t.Errorf("err %v served %v, want a.log", err, specPaths(specs))
		}
	})
	t.Run("NoIgnore walks every regular file", func(t *testing.T) {
		root := ignoreTree(t, true)
		var sk WalkSkipped
		specs, _, _ := Walk(root, nil, WalkOptions{Pattern: needle, NoIgnore: true, Skipped: &sk})
		if len(specs) != 7 || sk != (WalkSkipped{}) {
			t.Errorf("served %v skipped %+v, want all 7 matching files and nothing skipped", specPaths(specs), sk)
		}
	})
	t.Run("outside a checkout .gitignore is not applied", func(t *testing.T) {
		root := ignoreTree(t, false)
		var sk WalkSkipped
		specs, _, _ := Walk(root, nil, WalkOptions{Pattern: needle, Skipped: &sk})
		if len(specs) != 6 || sk.Binary != 1 || sk.Ignored != 0 || sk.IgnoredDirs != 0 {
			t.Errorf("served %v skipped %+v, want 6 files and only the binary skipped", specPaths(specs), sk)
		}
	})
}

// TestTheIgnoreMatcherFoldsCaseWhereGitDoes: git init sets core.ignorecase by
// probing the filesystem, and matches ignore rules without regard to case where
// it is true. foldsCase must give git's answer, and a walk must then skip A.LOG
// for `*.log` and gen/f for `Gen/`, as git ls-files does.
func TestTheIgnoreMatcherFoldsCaseWhereGitDoes(t *testing.T) {
	ig := staticIgnorer(parseIgnore("", []byte("*.log\nGen/\n"), true))
	if !ig.Ignored("A.LOG", false, 0) || !ig.Ignored("gen", true, 0) || ig.Ignored("a.txt", false, 0) {
		t.Error("a folding rule set does not ignore A.LOG and gen/ for *.log and Gen/")
	}
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git on PATH: the oracle for core.ignorecase")
	}
	root := t.TempDir()
	if out, err := exec.Command(git, "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Skipf("git init failed: %v %s", err, out)
	}
	out, _ := exec.Command(git, "-C", root, "config", "--bool", "core.ignorecase").Output()
	gitFolds := strings.TrimSpace(string(out)) == "true"
	if got := foldsCase(root); got != gitFolds {
		t.Fatalf("foldsCase says %v, git init set core.ignorecase to %v", got, gitFolds)
	}
	for n, b := range map[string]string{".gitignore": "*.log\nGen/\n", "A.LOG": "needle\n", "gen/f": "needle\n", "keep": "needle\n"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.Dir(n)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, n), []byte(b), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	specs, _, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("needle")})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, s := range specs {
		got = append(got, s.Path)
	}
	want := []string{"A.LOG", "gen/f", "keep"}
	if gitFolds {
		want = []string{"keep"}
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("core.ignorecase %v: the walk served %v, want %v", gitFolds, got, want)
	}
}

// ADR-116 T1. The matcher is a reimplementation, so it is held to git's own
// answer over a fixture tree: git is a test oracle here, never a dependency.
func TestTheIgnoreMatcherAgreesWithGit(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git on PATH: the cross-check needs it as an oracle")
	}
	root := t.TempDir()
	if out, err := exec.Command(git, "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Skipf("git init failed: %v %s", err, out)
	}
	files := map[string]string{
		".gitignore":     "*.log\n!keep.log\nbuild/\n/top.txt\nd/x.go\n**/gen\na/**/b\nf?.txt\n[ab].c\n\\#h\nsp\\ \n",
		"sub/.gitignore": "*.tmp\n/y\n!z.log\n",
	}
	paths := []string{"a.log", "keep.log", "build/o", "top.txt", "d/top.txt", "d/x.go", "e/d/x.go", "gen/q", "p/gen/q",
		"a/b", "a/x/b", "f1.txt", "f12.txt", "a.c", "c.c", "#h", "sp ", "sub/x.tmp", "x.tmp", "sub/y", "sub/w/y", "sub/z.log", "ok.go"}
	for n, b := range files {
		if err := os.WriteFile(filepath.Join(root, n), []byte(b), 0o644); err != nil {
			if err := os.MkdirAll(filepath.Join(root, filepath.Dir(n)), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, n), []byte(b), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, p := range paths {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(git, "-C", root, "check-ignore", "--no-index", "--stdin")
	cmd.Stdin = strings.NewReader(strings.Join(paths, "\n") + "\n")
	var out bytes.Buffer
	cmd.Stdout = &out
	_ = cmd.Run() // exit 1 means none ignored; the output is the verdict
	gitSays := map[string]bool{}
	for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if l != "" {
			gitSays[l] = true
		}
	}
	ig := newIgnorer(root)
	if ig == nil {
		t.Fatal("a git checkout was not recognised as one")
	}
	for _, p := range paths {
		if got := ig.Ignored(p, false, 0); got != gitSays[p] {
			t.Errorf("%q: matcher %v, git %v", p, got, gitSays[p])
		}
	}
}

// plant writes files (path → content, "/"-joined) under root.
func plant(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for n, b := range files {
		p := filepath.Join(root, filepath.FromSlash(n))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(b), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestAnIgnoredDirectoryYouNameIsWalked: naming a directory .gitignore
// ignores walks it, as naming one --exclude matches does; rules still apply
// to what is inside it, and the walk from the root still prunes it.
func TestAnIgnoredDirectoryYouNameIsWalked(t *testing.T) {
	root := t.TempDir()
	plant(t, root, map[string]string{
		".git/HEAD":  "x\n",
		".gitignore": "gen/\n*.log\n",
		"gen/a.go":   "needle\n",
		"gen/s/b.go": "needle\n",
		"gen/c.log":  "needle\n",
		"keep.go":    "needle\n",
	})
	needle := regexp.MustCompile("needle")
	for _, named := range []string{"gen", "gen/", "gen/s"} {
		var sk WalkSkipped
		specs, _, _ := Walk(root, []string{named}, WalkOptions{Pattern: needle, Skipped: &sk})
		want := "gen/a.go,gen/s/b.go"
		if named == "gen/s" {
			want = "gen/s/b.go"
		}
		if got := strings.Join(specPaths(specs), ","); got != want {
			t.Errorf("named %q: served %s, want %s", named, got, want)
		}
		if named == "gen" && sk.Ignored != 1 {
			t.Errorf("named gen: skipped %+v, want gen/c.log counted", sk)
		}
	}
	specs, _, _ := Walk(root, nil, WalkOptions{Pattern: needle})
	if got := strings.Join(specPaths(specs), ","); got != "keep.go" {
		t.Errorf("from the root: served %s, want keep.go alone", got)
	}
}

// TestAWorktreeReadsTheCommonInfoExclude: in a worktree .git is a file naming
// a gitdir whose commondir holds info/exclude, and git applies it there.
func TestAWorktreeReadsTheCommonInfoExclude(t *testing.T) {
	base := t.TempDir()
	plant(t, base, map[string]string{
		"main/.git/info/exclude":           "secret.txt\n",
		"main/.git/worktrees/w/commondir":  "../..\n",
		"main/.git/modules/m/info/exclude": "mod.txt\n",
		"w/.git":                           "gitdir: ../main/.git/worktrees/w\n",
		"w/secret.txt":                     "needle\n",
		"w/mod.txt":                        "needle\n",
		"m/.git":                           "gitdir: ../main/.git/modules/m\n",
		"m/secret.txt":                     "needle\n",
		"m/mod.txt":                        "needle\n",
	})
	needle := regexp.MustCompile("needle")
	for dir, want := range map[string]string{"w": "mod.txt", "m": "secret.txt"} {
		specs, _, _ := Walk(filepath.Join(base, dir), nil, WalkOptions{Pattern: needle})
		if got := strings.Join(specPaths(specs), ","); got != want {
			t.Errorf("%s: served %s, want %s", dir, got, want)
		}
	}
}

// staticIgnorer is an ignorer over rules already parsed, reading nothing.
func staticIgnorer(rules []ignoreRule) *ignorer {
	return &ignorer{rules: rules, static: true}
}
