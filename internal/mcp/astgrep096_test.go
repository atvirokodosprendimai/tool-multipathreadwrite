package mcp

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

// ADR-096 T2 over MCP ast_grep: a named path outside the root is refused
// before the binary starts, so it is never searched.
func TestMcpAstGrepRefusesANamedPathOutsideTheRoot(t *testing.T) {
	root, _ := checkout(t, "a.go", "package a\nfunc Target() {}\n")
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "x.go"), []byte("package x\nfunc Target() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	argv := installRecordingAstGrep(t, `[{"file":"a.go","range":{"start":{"line":1},"end":{"line":1}}}]`)
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{outside}, "ast_grep": "func Target"})
	all := fmt.Sprint(res["content"])
	if res["isError"] != true {
		t.Errorf("a named path outside the root alone is not isError: %v", res)
	}
	if !strings.Contains(all, "outside the root") {
		t.Errorf("the answer does not say outside the root:\n%s", all)
	}
	if _, statErr := os.Stat(argv); statErr == nil {
		t.Error("ast-grep ran on a path outside the root")
	}
}
