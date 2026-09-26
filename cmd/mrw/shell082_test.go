package main

import (
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
)

// needShell skips a test whose check must start when this machine has no POSIX
// shell: no sh on PATH and, on Windows, no Git for Windows (ADR-082). The test
// is about the check's verdict, and without a shell there is no check to judge.
func needShell(t *testing.T) {
	t.Helper()
	if _, ok := check.Shell(); !ok {
		t.Skip("no POSIX shell: no sh on PATH and no Git for Windows (ADR-082)")
	}
}

// ADR-082, the review of #247. A check that could not start for want of a shell
// was told to "declare one in .quality-harness.json" — one was declared, and
// declaring another changes nothing. The report names the missing shell and
// stops there; the pair, no check declared at all, still says to declare one.
func TestAMissingShellIsNotToldToDeclareACheck(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\n",
		".quality-harness.json": `{"check":"exit 0"}`,
	})
	t.Setenv("PATH", t.TempDir())
	out, code := runIn(t, root, "check", "a.go")
	if code != exitUsage || !strings.Contains(out, check.NoShell) || strings.Contains(out, "declare one") {
		t.Errorf("exit %d, want the missing shell named and no advice to declare a check:\n%s", code, out)
	}
	if got := declareAdvice("could not start: nothing declared"); !strings.Contains(got, "declare one") {
		t.Errorf("a missing check lost its advice: %q", got)
	}
}
