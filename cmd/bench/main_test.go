package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestTheMineFloorIsAFlag: ADR-160 Decision 2 raises the wide-rename floor to
// 25 occurrences across 12 files when A is at ceiling, and the redraw is made
// by bench mine itself (T6). Without the flags the floor is T2's, 15 and 8.
func TestTheMineFloorIsAFlag(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bench runs on the operator's macOS or Linux machine")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH")
	}
	repo := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/m\n\ngo 1.21\n",
		"a/a.go": "package a\n\n// Thing does nothing.\nfunc Thing() {}\n",
		"b/b.go": "package b\n\nimport \"example.com/m/a\"\n\n// B calls a.Thing twice.\nfunc B() {\n\ta.Thing()\n\ta.Thing()\n}\n",
	}
	for name, body := range files {
		p := filepath.Join(repo, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git := func(args ...string) string {
		out, err := exec.Command("git", append([]string{"-C", repo, "-c", "user.name=b", "-c", "user.email=b@b"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("add", "-A")
	git("commit", "-qm", "base")
	sha := git("rev-parse", "HEAD")
	list := func(extra ...string) string {
		var out bytes.Buffer
		if err := mine(append([]string{"--kind", "wide-rename", "--repo", repo, "--commit", sha}, extra...), &out); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	if got := list(); strings.Contains(got, "Thing") {
		t.Errorf("under the default floor (15, 8) Thing is listed: %q", got)
	}
	if got := list("--min-occurrences", "3", "--min-files", "2"); !strings.HasPrefix(got, "Thing\t") {
		t.Errorf("with --min-occurrences 3 --min-files 2 Thing is not listed: %q", got)
	}
	if got := list("--min-occurrences", "3", "--min-files", "3"); strings.Contains(got, "Thing") {
		t.Errorf("with --min-files 3 Thing, in 2 files, is listed: %q", got)
	}
}
