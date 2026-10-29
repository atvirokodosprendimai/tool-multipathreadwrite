package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	thenShOffMsg = "--then-sh runs any shell command, so it is off unless MRW_SHELL=on is in mrw's environment; a step the project declares in .quality-harness.json runs with --then NAME"
	runOffMsg    = "mrw run runs any shell command, so it is off unless MRW_SHELL=on is in mrw's environment"
)

// shell174Case is one way to ask mrw to run a command the caller wrote.
type shell174Case struct {
	name string
	argv func(plan string) []string
	msg  string
}

var shell174Cases = []shell174Case{
	{"write --then-sh", func(p string) []string { return []string{"write", "--then-sh", "touch marker", p} }, thenShOffMsg},
	{"write --then-sh then --then", func(p string) []string {
		return []string{"write", "--then-sh", "touch marker", "--then", "a", p}
	}, thenShOffMsg},
	{"write --then then --then-sh", func(p string) []string {
		return []string{"write", "--then", "a", "--then-sh", "touch marker", p}
	}, thenShOffMsg},
	{"check --then-sh", func(string) []string { return []string{"check", "--then-sh", "touch marker", "a.go"} }, thenShOffMsg},
	{"check --then-sh then --then", func(string) []string {
		return []string{"check", "--then-sh", "touch marker", "--then", "a", "a.go"}
	}, thenShOffMsg},
	{"check --then then --then-sh", func(string) []string {
		return []string{"check", "--then", "a", "--then-sh", "touch marker", "a.go"}
	}, thenShOffMsg},
	{"run", func(string) []string { return []string{"run", "touch marker"} }, runOffMsg},
}

// shell174Tree is a tree, its plan and a state home, with nothing run yet.
func shell174Tree(t *testing.T) (root, plan, state string) {
	t.Helper()
	state = t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	root = stepsTree(t, "touch checked")
	return root, primed(t, root), state
}

// shell174Nothing fails the test if anything a refused call must not do was
// done: a plan landed, a check or step ran, a marker was made, or mrw made a
// run log directory or kept a refused plan (the state home holds no mrw-*).
func shell174Nothing(t *testing.T, what, root, state, out string) {
	t.Helper()
	if b, _ := os.ReadFile(filepath.Join(root, "a.go")); string(b) != "package a\nfunc A() {}\n" {
		t.Errorf("%s: a.go changed:\n%s", what, out)
	}
	for _, n := range []string{"marker", "log", "checked"} {
		if _, err := os.Stat(filepath.Join(root, n)); err == nil {
			t.Errorf("%s: %s exists, something ran:\n%s", what, n, out)
		}
	}
	ents, _ := filepath.Glob(filepath.Join(state, "mrw-*"))
	if len(ents) != 0 {
		t.Errorf("%s: mrw made %v in the state home:\n%s", what, ents, out)
	}
}

// ADR-174 T1, the record's Enforced-by. --then-sh on write and check, and mrw
// run, run a command the caller wrote only when MRW_SHELL is exactly on; else
// they are refused, exit 2, naming the variable, before anything is written,
// run or logged. The refusal comes after today's depth, empty-command,
// placeholder and usage refusals; --then NAME and the declared check are
// unchanged.
func TestAdHocShellRunsOnlyWhenTheEnvironmentEnablesIt(t *testing.T) {
	needShell(t)

	for _, v := range []string{"", "yes", "ON", " on", "on ", "off"} {
		for _, c := range shell174Cases {
			t.Run("refused with MRW_SHELL="+v+": "+c.name, func(t *testing.T) {
				t.Setenv("MRW_SHELL", v)
				root, plan, state := shell174Tree(t)
				out, code := runIn(t, root, c.argv(plan)...)
				if code != exitUsage || !strings.Contains(out, c.msg) {
					t.Errorf("exit %d, want 2 with %q:\n%s", code, c.msg, out)
				}
				suffix := `(MRW_SHELL is "` + v + `"; only the exact value on enables it)`
				if has := strings.Contains(out, suffix); has != (v != "") {
					t.Errorf("the value suffix %q present %v, want %v:\n%s", suffix, has, v != "", out)
				}
				shell174Nothing(t, c.name, root, state, out)
			})
		}
	}

	for _, c := range shell174Cases {
		t.Run("enabled with MRW_SHELL=on: "+c.name, func(t *testing.T) {
			t.Setenv("MRW_SHELL", "on")
			root, plan, _ := shell174Tree(t)
			out, code := runIn(t, root, c.argv(plan)...)
			if code != 0 {
				t.Fatalf("exit %d, want 0:\n%s", code, out)
			}
			if _, err := os.Stat(filepath.Join(root, "marker")); err != nil {
				t.Errorf("the command did not run: %v\n%s", err, out)
			}
			if strings.Contains(c.name, "--then then") || strings.Contains(c.name, "then --then") {
				if got := logOf(t, root); got != "a\n" {
					t.Errorf("the declared step ran %q, want a once", got)
				}
			}
		})
	}

	t.Run("refusals under --json are one document", func(t *testing.T) {
		t.Setenv("MRW_SHELL", "")
		root, plan, state := shell174Tree(t)
		for _, c := range shell174Cases {
			argv := append([]string{c.argv(plan)[0], "--json"}, c.argv(plan)[1:]...)
			r, code, out := jsonDoc(t, root, argv...)
			if code != exitUsage || !strings.Contains(r.Error, c.msg) {
				t.Errorf("%s --json: exit %d, error %q:\n%s", c.name, code, r.Error, out)
			}
			shell174Nothing(t, c.name+" --json", root, state, out)
		}
	})

	t.Run("the older refusals answer first", func(t *testing.T) {
		t.Setenv("MRW_SHELL", "")
		root, plan, _ := shell174Tree(t)
		for _, c := range []struct {
			name string
			argv []string
			want string
		}{
			{"placeholder on write", []string{"write", "--then-sh", "echo {files}", plan}, "{files}"},
			{"placeholder on check", []string{"check", "--then-sh", "echo {packages}", "a.go"}, "{packages}"},
			{"empty command on write", []string{"write", "--then-sh", " ", plan}, "needs a command"},
			{"empty command on check", []string{"check", "--then-sh", " ", "a.go"}, "needs a command"},
			{"two arguments to run", []string{"run", "touch marker", "x"}, "exactly one argument"},
		} {
			out, code := runIn(t, root, c.argv...)
			if code != exitUsage || !strings.Contains(out, c.want) || strings.Contains(out, "MRW_SHELL") {
				t.Errorf("%s: exit %d, want 2 naming %q and not the switch:\n%s", c.name, code, c.want, out)
			}
		}
		t.Setenv("MRW_STEP_DEPTH", "8")
		for _, c := range shell174Cases {
			out, code := runIn(t, root, c.argv(plan)...)
			if code != exitUsage || !strings.Contains(out, "MRW_STEP_DEPTH") || strings.Contains(out, "MRW_SHELL") {
				t.Errorf("%s at depth 8: exit %d, want the depth refusal and not the switch:\n%s", c.name, code, out)
			}
		}
	})

	t.Run("a declared step and the declared check need no switch", func(t *testing.T) {
		t.Setenv("MRW_SHELL", "")
		root, plan, _ := shell174Tree(t)
		out, code := runIn(t, root, "write", "--then", "a", plan)
		if code != 0 || logOf(t, root) != "a\n" {
			t.Errorf("write --then a: exit %d, log %q:\n%s", code, logOf(t, root), out)
		}
		if _, err := os.Stat(filepath.Join(root, "checked")); err != nil {
			t.Errorf("the declared check did not run after a code write: %v", err)
		}
		out, code = runIn(t, root, "check", "--then", "b", "a.go")
		if code != 0 || logOf(t, root) != "a\nb\n" {
			t.Errorf("check --then b: exit %d, log %q:\n%s", code, logOf(t, root), out)
		}
	})

	t.Run("a nested mrw inherits the variable, and loses it with the environment", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("env -u is POSIX")
		}
		bin := buildMrw(t)
		state := t.TempDir()
		root := t.TempDir()
		env := []string{"XDG_STATE_HOME=" + state, "MRW_SHELL=on"}
		inner := posixQuote(bin) + " -C " + posixQuote(root) + " run 'touch nested'"
		if _, errOut, code := runBin(t, bin, env, "-C", root, "run", inner); code != 0 {
			t.Fatalf("nested run: exit %d: %s", code, errOut)
		}
		if _, err := os.Stat(filepath.Join(root, "nested")); err != nil {
			t.Errorf("the nested run did not inherit the switch: %v", err)
		}
		dropped := "env -u MRW_SHELL " + strings.Replace(inner, "'touch nested'", "'touch dropped'", 1)
		if out, _, code := runBin(t, bin, env, "-C", root, "run", dropped); code != exitUsage || !strings.Contains(out, "exit 2") {
			t.Errorf("a nested run without the variable: exit %d, want the inner refusal's 2:\n%s", code, out)
		}
		if _, err := os.Stat(filepath.Join(root, "dropped")); err == nil {
			t.Error("a nested run ran without the variable")
		}
	})
}
