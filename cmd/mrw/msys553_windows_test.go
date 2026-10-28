//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
)

// Issue #553, through the shell that does the rewriting: Git for Windows' sh,
// the one a check runs under (ADR-082). The advised prefix serves a regex
// address, the bare form is rewritten and gets the hint, and the prefix stays
// on its one command — the next command in the same shell is rewritten again.
// It never skips: a runner without Git's sh cannot assert this, and saying so
// is the finding.
func TestTheMSYSPrefixFormWorksAndStaysOnItsCommand(t *testing.T) {
	sh, pathDir, ok := check.Shell()
	if !ok {
		t.Fatal("no Git for Windows sh: the MSYS advice cannot be asserted on this runner")
	}
	exe := mrwExe(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "f.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(script string) (string, int) {
		t.Helper()
		cmd := exec.Command(sh, "-c", script)
		cmd.Dir = root
		env := append(os.Environ(), "MRW_EXE="+exe, "XDG_STATE_HOME="+t.TempDir())
		if pathDir != "" {
			env = append(env, "PATH="+pathDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		}
		// The runner's own environment must not already switch conversion off.
		var clean []string
		for _, kv := range env {
			if !strings.HasPrefix(kv, "MSYS_NO_PATHCONV=") && !strings.HasPrefix(kv, "MSYS2_ARG_CONV_EXCL=") {
				clean = append(clean, kv)
			}
		}
		cmd.Env = clean
		out, err := cmd.CombinedOutput()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if err != nil {
			t.Fatalf("running %s failed outside the process's own status: %v", sh, err)
		}
		return string(out), code
	}
	served := func(out string) bool { return strings.Contains(out, "2| two") }

	for _, prefix := range []string{"MSYS_NO_PATHCONV=1", "MSYS2_ARG_CONV_EXCL='*'"} {
		if out, code := run(prefix + ` "$MRW_EXE" read 'sub/f.txt:/two/'`); code != 0 || !served(out) {
			t.Errorf("%s on the command: exit %d, want the regex address served:\n%s", prefix, code, out)
		}
	}
	out, code := run(`"$MRW_EXE" read 'sub/f.txt:/two/'`)
	if code == 0 || served(out) || !strings.Contains(out, "MSYS_NO_PATHCONV=1 mrw") {
		t.Errorf("the bare form: exit %d, want it rewritten and the hint naming the prefix:\n%s", code, out)
	}
	out, code = run(`MSYS_NO_PATHCONV=1 "$MRW_EXE" read 'sub/f.txt:/two/' >/dev/null && "$MRW_EXE" read 'sub/f.txt:/two/'`)
	if code == 0 || served(out) {
		t.Errorf("the command after a prefixed one: exit %d, want it rewritten again (the prefix leaked):\n%s", code, out)
	}
}
