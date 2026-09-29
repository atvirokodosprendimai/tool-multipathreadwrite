package read

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// ADR-096 T2. read.AstGrep judges every named path as read.Walk does before
// the binary starts: a refused path is a Problem and never reaches ast-grep,
// an accepted one reaches it as the absolute path mrw judged ("." for the
// root), and with nothing left the binary is not started at all. The fakes
// the older tests use print fixed JSON whatever they are asked, so they
// cannot tell what reached the binary; this one records its arguments.

// installRecordingAstGrep puts an ast-grep on PATH that appends each argument
// it is given, one per line, to the returned file before printing stdout. The
// warm-up start with no arguments records nothing, so the file exists only if
// mrw ran the binary.
func installRecordingAstGrep(t *testing.T, stdout string) string {
	t.Helper()
	dir := t.TempDir()
	argv := filepath.Join(dir, "argv")
	src := filepath.Join(dir, "fake.go")
	body := fmt.Sprintf(`package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) > 1 {
		f, err := os.OpenFile(%q, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			os.Exit(3)
		}
		fmt.Fprintln(f, strings.Join(os.Args[1:], "\n"))
		_ = f.Close()
	}
	fmt.Fprint(os.Stdout, %q)
}
`, argv, stdout)
	if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "ast-grep")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, src)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fake ast-grep: %v\n%s", err, out)
	}
	_ = exec.Command(bin).Run()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return argv
}

// astGrepOperands returns the path operands the recording fake was given —
// everything after --json — and whether it ran at all.
func astGrepOperands(t *testing.T, argv string) ([]string, bool) {
	t.Helper()
	b, err := os.ReadFile(argv)
	if os.IsNotExist(err) {
		return nil, false
	}
	if err != nil {
		t.Fatal(err)
	}
	all := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	for i, a := range all {
		if a == "--json" {
			return all[i+1:], true
		}
	}
	t.Fatalf("the fake's argv has no --json: %q", all)
	return nil, true
}

// hit096 is one ast-grep JSON hit on the first line of each file named.
func hit096(files ...string) string {
	var parts []string
	for _, f := range files {
		parts = append(parts, `{"file":"`+f+`","range":{"start":{"line":0},"end":{"line":0}}}`)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// realRoot096 is the root as the judge resolves it (on macOS, /var is
// /private/var), so an expected absolute operand is spelled as mrw spells it.
func realRoot096(t *testing.T, root string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func problemPaths096(probs []Problem) []string {
	var out []string
	for _, p := range probs {
		out = append(out, p.Path)
	}
	return out
}

func TestAstGrepNeverReceivesANamedPathTheBoundaryRefuses(t *testing.T) {
	outer := t.TempDir()
	root := filepath.Join(outer, "repo")
	for name, body := range map[string]string{
		"repo/a.go":           "Target\n",
		"repo/d/f.go":         "Target\n",
		"repo/.st/mrw/k/seen": "Target\n",
		"out/x.go":            "Target\n",
	} {
		full := filepath.Join(outer, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, ".st"))
	if err := os.Symlink("d", filepath.Join(root, "dlink")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink("loop", filepath.Join(root, "loop")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	refused := []string{"../out", "nosuch.go", "dlink", "loop", ".st/mrw"}
	if err := exec.Command("mkfifo", filepath.Join(root, "pipe")).Run(); err == nil {
		refused = append(refused, "pipe")
	}
	argv := installRecordingAstGrep(t, hit096("a.go"))
	specs, probs, err := AstGrep(root, append([]string{"a.go"}, refused...), "Target", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := paths(specs); !reflect.DeepEqual(got, []string{"a.go"}) {
		t.Errorf("served %v; want a.go's hit", got)
	}
	if got := problemPaths096(probs); !reflect.DeepEqual(got, refused) {
		t.Errorf("problems name %v; want one each for %v: %v", got, refused, probs)
	}
	ops, ran := astGrepOperands(t, argv)
	want := []string{filepath.Join(realRoot096(t, root), "a.go")}
	if !ran || !reflect.DeepEqual(ops, want) {
		t.Errorf("ast-grep was given %q (ran=%v); want only %q", ops, ran, want)
	}
}

func TestAstGrepDoesNotRunWhenEveryNamedPathIsRefused(t *testing.T) {
	outer := t.TempDir()
	root := filepath.Join(outer, "repo")
	for _, d := range []string{filepath.Join(root, "d"), filepath.Join(outer, "out")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outer, "out", "x.go"), []byte("Target\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("d", filepath.Join(root, "dlink")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	argv := installRecordingAstGrep(t, hit096("a.go"))
	specs, probs, err := AstGrep(root, []string{"../out", "dlink"}, "Target", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 0 {
		t.Errorf("served %v with every named path refused", paths(specs))
	}
	if got := problemPaths096(probs); !reflect.DeepEqual(got, []string{"../out", "dlink"}) {
		t.Errorf("problems name %v; want ../out and dlink: %v", got, probs)
	}
	if _, ran := astGrepOperands(t, argv); ran {
		t.Error("ast-grep ran although every named path was refused; with nothing left it searches .")
	}
}

func TestAstGrepHandsAPathThatIsTheRootOnAsDot(t *testing.T) {
	root := tree(t, map[string]string{"a.go": "Target\n"})
	if err := os.Symlink(".", filepath.Join(root, "self")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	linked := filepath.Join(t.TempDir(), "L")
	if err := os.Symlink(root, linked); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	for _, c := range []struct{ what, root, name string }{
		{"a relative self -> .", root, "self"},
		{"the root reached through a link, named absolutely", linked, linked},
	} {
		argv := installRecordingAstGrep(t, hit096("a.go"))
		specs, probs, err := AstGrep(c.root, []string{c.name}, "Target", nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(probs) != 0 || !reflect.DeepEqual(paths(specs), []string{"a.go"}) {
			t.Errorf("%s: served %v, problems %v; want a.go's hit", c.what, paths(specs), probs)
		}
		if ops, _ := astGrepOperands(t, argv); !reflect.DeepEqual(ops, []string{"."}) {
			t.Errorf("%s: ast-grep was given %q; want .", c.what, ops)
		}
	}
}

func TestAstGrepHandsOnTheAbsolutePathItJudged(t *testing.T) {
	outer := t.TempDir()
	root := filepath.Join(outer, "repo")
	for name, body := range map[string]string{"repo/f.go": "Target\n", "repo/-p": "Target\n", "repo/d/g.go": "Target\n", "f.go": "Other\n"} {
		full := filepath.Join(outer, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(".", filepath.Join(root, "self")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink("d", filepath.Join(root, "dlink")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	argv := installRecordingAstGrep(t, hit096("f.go", "-p", "d/g.go"))
	specs, probs, err := AstGrep(root, []string{"self/../f.go", "-p", filepath.Join(root, "dlink")}, "Target", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(probs) != 0 {
		t.Errorf("problems %v; every named path here is accepted", probs)
	}
	if got := paths(specs); !reflect.DeepEqual(got, []string{"-p", "d/g.go", "f.go"}) {
		t.Errorf("served %v; want -p, d/g.go and f.go", got)
	}
	real := realRoot096(t, root)
	want := []string{filepath.Join(real, "f.go"), filepath.Join(real, "-p"), filepath.Join(real, "d")}
	if ops, _ := astGrepOperands(t, argv); !reflect.DeepEqual(ops, want) {
		t.Errorf("ast-grep was given %q; want the judged absolute paths %q", ops, want)
	}
}

func TestAstGrepRefusesANamedPathItCannotOpen(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("mode 000 is not enforced here")
	}
	root := tree(t, map[string]string{"a.go": "Target\n", "secret.go": "Target\n", "locked/x.go": "Target\n"})
	for _, p := range []string{"secret.go", "locked"} {
		full := filepath.Join(root, p)
		if err := os.Chmod(full, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(full, 0o755) })
	}
	argv := installRecordingAstGrep(t, hit096("a.go"))
	specs, probs, err := AstGrep(root, []string{"a.go", "secret.go", "locked"}, "Target", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := paths(specs); !reflect.DeepEqual(got, []string{"a.go"}) {
		t.Errorf("served %v; want a.go's hit", got)
	}
	if got := problemPaths096(probs); !reflect.DeepEqual(got, []string{"secret.go", "locked"}) {
		t.Errorf("problems name %v; want secret.go and locked: %v", got, probs)
	}
	for _, p := range probs {
		if !strings.Contains(p.Reason, "permission denied") {
			t.Errorf("%s: reason %q does not give the OS's error", p.Path, p.Reason)
		}
	}
	want := []string{filepath.Join(realRoot096(t, root), "a.go")}
	if ops, _ := astGrepOperands(t, argv); !reflect.DeepEqual(ops, want) {
		t.Errorf("ast-grep was given %q; want only %q", ops, want)
	}
}

func TestARefusedNamedDirectoryIsNoAstGrepStart(t *testing.T) {
	root := tree(t, map[string]string{"d/f.go": "Target\n", "a.go": "Target\n"})
	if err := os.Symlink("d", filepath.Join(root, "dlink")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	installRecordingAstGrep(t, hit096("d/f.go"))
	specs, probs, err := AstGrep(root, []string{".", "dlink"}, "Target", []string{"d"})
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 0 {
		t.Errorf("served %v; --exclude d drops d/f.go once dlink is no start", paths(specs))
	}
	if got := problemPaths096(probs); !reflect.DeepEqual(got, []string{"dlink"}) {
		t.Errorf("problems name %v; want dlink alone: %v", got, probs)
	}
}
