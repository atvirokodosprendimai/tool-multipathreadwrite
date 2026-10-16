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
		{[]string{"read", "-"}, 0, []string{"==> -  5L", "    2| b"}, nil},
		{[]string{"read", "-:2"}, 0, []string{"@@ 2-3", "    2| b", "    3| c"}, []string{"    1| a"}},
		{[]string{"read", "--grep", "^c$", "-"}, 0, []string{"    3| c"}, []string{"    1| a"}},
		{[]string{"read", "--grep", "^b$", "-C", "1", "-"}, 0, []string{"    1| a", "    3| c"}, []string{"    5| "}},
		{[]string{"read", "--max-lines", "1", "-"}, 1, []string{"    1| a"}, []string{"    2| b"}},
		{[]string{"read", "--max-cols", "8", "-:5"}, 0, []string{"    5| "}, []string{strings.Repeat("x", 30)}},
		{[]string{"read", "-", "--grep", "^b$"}, 0, []string{"    2| b"}, []string{"    1| a"}},
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
	// CR-only match withheld by --max-lines is incomplete, not no match.
	if _, errOut, code := run("a\n", "read", "--grep", "^$", "-"); code != 1 || !strings.Contains(errOut, "mrw: no match:") {
		t.Errorf("^$ over a\\n: exit %d, want 1 no match:\n%s", code, errOut)
	}
	if _, errOut, code := run("a\rb\r", "read", "--grep", "^b$", "--max-lines", "0", "-"); code != 1 || !strings.Contains(errOut, "mrw: incomplete:") {
		t.Errorf("a withheld CR-only match: exit %d, want 1 incomplete:\n%s", code, errOut)
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
	// The refusals made before parsing say why, not a panic (Codex re-review of T8).
	for args, want := range map[string]string{
		"read - - --stat":    "after a lone -",
		"read - --max-lines": "needs a value",
		"read - ./-":         "after a lone -",
	} {
		if _, e, c := run(in, strings.Fields(args)...); c != 2 || !strings.Contains(e, want) || strings.Contains(e, "panic") {
			t.Errorf("mrw %s: exit %d, want 2 saying %q:\n%s", args, c, want, e)
		}
	}

	// The flag parser stops at a lone - and drops what follows, flags included:
	// `write - --dry-run` wrote the plan. A lone - moves last before parsing, so
	// every flag after it is honoured; a - that is a flag's value is not moved.
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

// ADR-161: the tokens that stop the parser — a lone -, a dash before a
// non-letter — move last, a lone - after the rest, so the flags after them are
// parsed; a flag's value (an inherited root flag's included), a token before the
// subcommand, one after --, and one already last stay. A second lone - is
// refused, and so is a move that would make - the value of a flag left without one.
func TestALoneDashMovesLast(t *testing.T) {
	root := rootCommand()
	for _, c := range []struct{ in, want string }{
		{"mrw write - --dry-run", "mrw write --dry-run -"},
		{"mrw write - --then vet --no-check", "mrw write --then vet --no-check -"},
		{"mrw -C dir read - --grep b", "mrw -C dir read --grep b -"},
		{"mrw --root dir read - -C 1", "mrw --root dir read -C 1 -"},
		{"mrw read -:2 --max-lines 1", "mrw read -:2 --max-lines 1"},
		{"mrw iter -1= clear", "mrw iter -1= clear"},
		{"mrw write --dry-run -", "mrw write --dry-run -"},
		{"mrw write --root - --dry-run", "mrw write --root - --dry-run"},
		{"mrw read --files-from - a.go", "mrw read --files-from - a.go"},
		{"mrw read --grep - x", "mrw read --grep - x"},
		{"mrw read -- - x", "mrw read -- - x"},
		{"mrw - read", "mrw - read"},
	} {
		got, err := liftLoneDash(root, strings.Fields(c.in))
		if err != nil || strings.Join(got, " ") != c.want {
			t.Errorf("%s: got %q (%v), want %q", c.in, strings.Join(got, " "), err, c.want)
		}
	}
	for _, in := range []string{"mrw read - - --stat", "mrw write --json - - --dry-run", "mrw write --dry-run - --create",
		"mrw read - -1= --stat", "mrw read - -- x", "mrw -- read - x", "mrw iter add - x"} {
		if _, err := liftLoneDash(root, strings.Fields(in)); err == nil {
			t.Errorf("%s: moved, want a refusal", in)
		}
	}
	// A padded ' - ' is not a lone -: ADR-069 T9's guard judges it where it stands.
	padded := []string{"mrw", "write", " - ", "--dry-run"}
	if got, err := liftLoneDash(root, padded); err != nil || strings.Join(got, "|") != strings.Join(padded, "|") {
		t.Errorf("a padded - was moved or refused: %q %v", got, err)
	}
}
