package main

import (
	"strings"
	"testing"
)

// ADR-161 Decision 11. `mrw read -` reads standard input as one unnamed file,
// with every address, --grep, -C, --max-lines and --max-cols, so a command's
// output is narrowed by mrw and not by grep, sed -n or head. It licenses
// nothing: it is not a file. A file named `-` is `./-`.
func TestReadingStandardInputLicensesNothing(t *testing.T) {
	root, run := subRun(t)
	in := "a\nb\nc\nd\n" + strings.Repeat("x", 30) + "c\n"
	for _, c := range []struct {
		args []string
		code int
		want []string
		not  []string
	}{
		{[]string{"read", "-"}, 0, []string{"==> -  5L", " 2| b"}, nil},
		{[]string{"read", "-:2"}, 0, []string{"@@ 2-3", " 2| b", " 3| c"}, []string{" 1| a"}},
		{[]string{"read", "--grep", "^c$", "-"}, 0, []string{" 3| c"}, []string{" 1| a"}},
		{[]string{"read", "--grep", "^b$", "-C", "1", "-"}, 0, []string{" 1| a", " 3| c"}, []string{" 5| "}},
		{[]string{"read", "--max-lines", "1", "-"}, 0, []string{" 1| a"}, []string{" 2| b"}},
		{[]string{"read", "--max-cols", "8", "-:5"}, 0, []string{" 5| "}, []string{strings.Repeat("x", 30)}},
		{[]string{"read", "-", "--grep", "^b$"}, 0, []string{" 2| b"}, []string{" 1| a"}},
	} {
		out, errOut, code := run(in, c.args...)
		name := strings.Join(c.args, " ")
		if code != c.code {
			t.Errorf("%s: exit %d, want %d:\n%s%s", name, code, c.code, out, errOut)
		}
		for _, w := range c.want {
			if !strings.Contains(out, w) {
				t.Errorf("%s: stdout lacks %q:\n%s%s", name, w, out, errOut)
			}
		}
		for _, n := range c.not {
			if strings.Contains(out, n) {
				t.Errorf("%s: stdout holds %q:\n%s", name, n, out)
			}
		}
	}
	if out, _, _ := run("", "seen"); !strings.Contains(out, "0 file(s) seen") {
		t.Errorf("a read of standard input recorded something:\n%s", out)
	}

	// A grep of standard input that matches nothing is `no match:`.
	if _, errOut, code := run("a\n", "read", "--grep", "z", "-"); code != 1 || !strings.Contains(errOut, "mrw: no match:") {
		t.Errorf("a grep of stdin matching nothing: exit %d, want 1 no match:\n%s", code, errOut)
	}
	// Lines are split as read.Run splits them: "a\n" has no empty line, and a
	// CR-only match withheld by --max-lines is capped (ADR-173), not no match.
	if _, errOut, code := run("a\n", "read", "--grep", "^$", "-"); code != 1 || !strings.Contains(errOut, "mrw: no match:") {
		t.Errorf("^$ over a\\n: exit %d, want 1 no match:\n%s", code, errOut)
	}
	if _, errOut, code := run("a\rb\r", "read", "--grep", "^b$", "--max-lines", "0", "-"); code != 0 || !strings.Contains(errOut, "mrw: capped:") {
		t.Errorf("a withheld CR-only match: exit %d, want 0 capped:\n%s", code, errOut)
	}

	// A file named `-` is ./-, read from disk and recorded like any file.
	subTree(t, root, map[string]string{"-": "on disk\n"})
	if out, errOut, code := run("from stdin\n", "read", "./-"); code != 0 || !strings.Contains(out, "on disk") || strings.Contains(out, "from stdin") {
		t.Errorf("read ./-: exit %d, want the file named -:\n%s%s", code, out, errOut)
	}

	for _, args := range [][]string{
		{"read", "--grep", "b", "--sub", "B", "-"},
		{"read", "--quiet", "-"},
		{"read", "--stat", "-"},
		{"read", "--count", "--grep", "b", "-"},
		{"read", "--files-from", "-", "-"},
		{"read", "-", "./-"},
		{"read", "--grep", "b", "-:1", "-"},
		{"read", "-C", "1", "-"},
		{"read", "-C", "1", "-:1"},
		{"read", "-", "-", "--stat"},
		{"read", "-", "--max-lines"},
	} {
		if o, e, c := run(in, args...); c != 2 || o != "" {
			t.Errorf("mrw %s: exit %d stdout %q, want 2 and nothing:\n%s", strings.Join(args, " "), c, o, e)
		}
	}
	// The refusals say why, not a panic (Codex re-review of T8). Since ADR-196
	// they are read's own: v3.14.0 parses on past a lone -.
	for args, want := range map[string]string{
		"read - - --stat":    "names nothing else",
		"read - --max-lines": "flag needs an argument: --max-lines",
		"read - ./-":         "names nothing else",
	} {
		if _, e, c := run(in, strings.Fields(args)...); c != 2 || !strings.Contains(e, want) || strings.Contains(e, "panic") {
			t.Errorf("mrw %s: exit %d, want 2 saying %q:\n%s", args, c, want, e)
		}
	}

	// urfave/cli v3.14.0 keeps a lone - and parses on, so every flag after it
	// is honoured: `write - --dry-run` is a dry run (ADR-196; v3.11.0 dropped
	// the flag, which ADR-161's lift worked around).
	subTree(t, root, map[string]string{"w.txt": "one\n"})
	if _, e, c := run("", "read", "w.txt"); c != 0 {
		t.Fatalf("read w.txt: exit %d\n%s", c, e)
	}
	if o, e, c := run("@@ w.txt 1 replace\nONE\n", "write", "-", "--dry-run"); c != 0 || subFile(t, root, "w.txt") != "one\n" || !strings.Contains(o, "dry run") {
		t.Errorf("write - --dry-run: exit %d, w.txt %q; want 0, a dry run and nothing written:\n%s%s", c, subFile(t, root, "w.txt"), o, e)
	}
	if o, e, c := run("w.txt\n", "read", "--files-from", "-"); c != 0 || !strings.Contains(o, "one") {
		t.Errorf("read --files-from -: exit %d, want the file listed on stdin:\n%s%s", c, o, e)
	}
}

// ADR-161 T8, retargeted by ADR-196 (the name is the done task's): the lift
// is gone, since urfave/cli v3.14.0 keeps a lone - and reads the flags after
// it. They are honoured where they stand, and a path after - is refused by
// the command's own rule, writing nothing.
func TestALoneDashMovesLast(t *testing.T) {
	root, run := subRun(t)
	subTree(t, root, map[string]string{"w.txt": "one\n"})
	if _, e, c := run("", "read", "w.txt"); c != 0 {
		t.Fatalf("read w.txt: exit %d\n%s", c, e)
	}
	if o, e, c := run("@@ w.txt 1 replace\nONE\n", "write", "-", "--dry-run", "--terse"); c != 0 || subFile(t, root, "w.txt") != "one\n" || !strings.Contains(o, "dry") {
		t.Errorf("write - --dry-run --terse: exit %d, w.txt %q:\n%s%s", c, subFile(t, root, "w.txt"), o, e)
	}
	if o, e, c := run("@@ w.txt 1 replace\nONE\n", "write", "-", "w.txt"); c != 2 || subFile(t, root, "w.txt") != "one\n" {
		t.Errorf("write - w.txt: exit %d, w.txt %q; want 2 and nothing written:\n%s%s", c, subFile(t, root, "w.txt"), o, e)
	}
}
