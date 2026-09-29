package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// installRecordingAstGrep puts an ast-grep on PATH that appends each argument
// it is given to the returned file before printing stdout — internal/read's
// helper, copied because test helpers do not cross packages. The warm-up start
// with no arguments records nothing, so the file exists only if mrw ran it.
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

// ADR-096 T2. A path outside the root named to --ast-grep was handed to the
// binary, which searched it; each matching file came back as a REFUSED line
// naming it. It is now refused before ast-grep starts, and never reaches it.
func TestAstGrepRefusesANamedPathOutsideTheRootBeforeRunning(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\nfunc Target() {}\n"})
	outside := grepTree(t, map[string]string{"x.go": "package x\nfunc Target() {}\n"})
	argv := installRecordingAstGrep(t, `[{"file":"a.go","range":{"start":{"line":1},"end":{"line":1}}}]`)
	out, err := readIn(t, root, "--ast-grep", "func Target", outside)
	if err == nil {
		t.Fatalf("a named path outside the root exited 0:\n%s", out)
	}
	if !strings.Contains(out, "REFUSED") || !strings.Contains(out, "outside the root") {
		t.Errorf("no REFUSED line saying outside the root:\n%s", out)
	}
	if _, statErr := os.Stat(argv); statErr == nil {
		t.Error("ast-grep ran on a path outside the root")
	}
	out, err = readIn(t, root, "--ast-grep", "func Target", "a.go")
	if err != nil {
		t.Fatalf("--ast-grep over a.go failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "==> a.go") {
		t.Errorf("a.go's hit was not served:\n%s", out)
	}
}
