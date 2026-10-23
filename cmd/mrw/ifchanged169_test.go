package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
)

// ADR-169 T1, through the built binary: the guard's refusals are exit 2 with
// nothing served, a walked guard the walk did not reach is named and not
// refused, and --max-chars and --json treat the unchanged line as output like
// any other (ADR-165).

// ifChangedRun builds mrw once for the test and returns a runner over a tree
// and a state directory of its own.
func ifChangedRun(t *testing.T, files map[string]string) (run func(stdin string, argv ...string) (string, string, int), tree string) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "mrw.exe") // .exe: Windows exec will not launch an extensionless file
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	tree, state := t.TempDir(), t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(tree, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return func(stdin string, argv ...string) (string, string, int) {
		t.Helper()
		cmd := exec.Command(bin, argv...)
		cmd.Dir = tree
		cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+state)
		cmd.Stdin = strings.NewReader(stdin)
		var o, e bytes.Buffer
		cmd.Stdout, cmd.Stderr = &o, &e
		code := 0
		var exit *exec.ExitError
		if err := cmd.Run(); errors.As(err, &exit) {
			code = exit.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		return o.String(), e.String(), code
	}, tree
}

var headerSha = regexp.MustCompile(`sha ([0-9a-f]{8})`)

func TestAGuardTheReadCannotUseIsRefused(t *testing.T) {
	run, _ := ifChangedRun(t, map[string]string{
		"a.txt":   "alpha\nbravo\ncharlie\n",
		"b.txt":   "kilo\nlima\nmike\n",
		"k=v.txt": "equals\n",
	})
	shaOf := func(name string) string {
		t.Helper()
		out, _, code := run("", "read", "--stat", name)
		m := headerSha.FindStringSubmatch(out)
		if code != 0 || m == nil {
			t.Fatalf("--stat %s: exit %d, %q", name, code, out)
		}
		return m[1]
	}
	a, b, kv := shaOf("a.txt"), shaOf("b.txt"), shaOf("k=v.txt")
	unchangedA := fmt.Sprintf("==> a.txt  unchanged since sha %s — 3 lines, 20 bytes not served\n", a)

	// Refused: exit 2, nothing served.
	for name, argv := range map[string][]string{
		"a named file no spec serves":     {"read", "--if-changed", "b.txt=" + b, "a.txt"},
		"two guards on one file":          {"read", "--if-changed", "a.txt=" + a, "--if-changed", "a.txt=" + a, "a.txt"},
		"two spellings of one file":       {"read", "--if-changed", "a.txt=" + a, "--if-changed", "./a.txt=" + a, "a.txt"},
		"a prefix shorter than 8":         {"read", "--if-changed", "a.txt=" + a[:7], "a.txt"},
		"a prefix longer than 64":         {"read", "--if-changed", "a.txt=" + strings.Repeat("a", 65), "a.txt"},
		"a prefix that is not hex":        {"read", "--if-changed", "a.txt=zzzzzzzz", "a.txt"},
		"no sha at all":                   {"read", "--if-changed", "a.txt", "a.txt"},
		"beside --stat":                   {"read", "--if-changed", "a.txt=" + a, "--stat", "a.txt"},
		"beside --quiet":                  {"read", "--if-changed", "a.txt=" + a, "--quiet", "a.txt"},
		"beside --count":                  {"read", "--if-changed", "a.txt=" + a, "--grep", "alpha", "--count", "."},
		"beside --sub":                    {"read", "--if-changed", "a.txt=" + a, "--grep", "alpha", "--sub", "x", "."},
		"beside --rewrite":                {"read", "--if-changed", "a.txt=" + a, "--ast-grep", "alpha", "--rewrite", "x", "."},
		"a guard on standard input":       {"read", "--if-changed", "a.txt=" + a, "-"},
		"a guard on a file past the root": {"read", "--if-changed", "../x.txt=" + a, "a.txt"},
	} {
		out, errOut, code := run("alpha\n", argv...)
		if code != 2 || out != "" {
			t.Errorf("%s: exit %d, stdout %q, stderr %q; want exit 2 and nothing served", name, code, out, errOut)
		}
	}

	// PATH is split from SHA at the last "=".
	if out, _, code := run("", "read", "--if-changed", "k=v.txt="+kv, "k=v.txt"); code != 0 ||
		out != fmt.Sprintf("==> k=v.txt  unchanged since sha %s — 1 lines, 7 bytes not served\n", kv) {
		t.Errorf("a path holding =: exit %d, %q", code, out)
	}

	// Two spellings of one file, one guard: both are the one file.
	if out, _, code := run("", "read", "--if-changed", "./a.txt="+a, "a.txt"); code != 0 || out != unchangedA {
		t.Errorf("a guard spelled ./a.txt on a.txt: exit %d, %q", code, out)
	}

	// Under a walk, a guard the walk did not reach is named, not refused; one it
	// reached is judged. Neither is a problem.
	out, errOut, code := run("", "read", "--grep", "alpha", "--if-changed", "b.txt="+b, "--if-changed", "a.txt="+a, ".")
	// The walk's spec is the matching line and the line after it: 2 lines, 12 bytes.
	if code != 0 || out != fmt.Sprintf("==> a.txt  unchanged since sha %s — 2 lines, 12 bytes not served\n", a) || !strings.Contains(errOut, "-- guard not used: b.txt (this read did not serve it)\n") ||
		strings.Contains(errOut, "incomplete") || strings.Contains(errOut, "no match") {
		t.Errorf("a walk: exit %d, stdout %q, stderr %q", code, out, errOut)
	}
	// A walk that matches no file at all still names the guard, and exits as without it.
	out, errOut, code = run("", "read", "--grep", "nomatchanywhere", "--if-changed", "b.txt="+b, ".")
	if code != 1 || out != "" || !strings.Contains(errOut, "-- guard not used: b.txt") || !strings.Contains(errOut, "no match") {
		t.Errorf("a walk that matched nothing: exit %d, stdout %q, stderr %q", code, out, errOut)
	}

	// --max-chars follows ADR-165: the unchanged line is output like any other.
	n := utf8.RuneCountInString(unchangedA)
	for _, c := range []struct {
		budget     int
		argv       []string
		code       int
		stdout     string
		withheldIn string
	}{
		{0, []string{"a.txt"}, 1, "", "a.txt: WITHHELD: --max-chars reached"},
		{n - 1, []string{"a.txt"}, 1, "", "a.txt: WITHHELD: --max-chars reached"},
		{n, []string{"a.txt"}, 0, unchangedA, ""},
		{n, []string{"a.txt", "b.txt"}, 1, unchangedA, "b.txt: WITHHELD: --max-chars reached"},
	} {
		argv := append([]string{"read", "--if-changed", "a.txt=" + a, "--max-chars", fmt.Sprint(c.budget)}, c.argv...)
		out, errOut, code := run("", argv...)
		if code != c.code || out != c.stdout || !strings.Contains(errOut, c.withheldIn) || (c.code == 1) != strings.Contains(errOut, "mrw: incomplete: ") {
			t.Errorf("--max-chars %d over %v: exit %d, stdout %q, stderr %q", c.budget, c.argv, code, out, errOut)
		}
		// The receipt lists only what reached the caller.
		jout, _, _ := run("", append([]string{"read", "--json"}, argv[1:]...)...)
		var doc struct {
			Receipt map[string]json.RawMessage `json:"receipt"`
		}
		if err := json.Unmarshal([]byte(jout), &doc); err != nil {
			t.Fatalf("--json: %v\n%s", err, jout)
		}
		_, listed := doc.Receipt["unchanged"]
		if want := strings.Contains(c.stdout, "unchanged since"); listed != want {
			t.Errorf("--max-chars %d over %v: unchanged listed = %v, want %v", c.budget, c.argv, listed, want)
		}
	}

	// A budget that fits the unchanged line exactly, before a missed range's
	// note that does not fit: the answer reached the caller, so the receipt
	// lists it, and the missed range still exits 1 (the Codex review of #514).
	zero := fmt.Sprintf("==> a.txt  unchanged since sha %s — 0 lines, 0 bytes not served\n", a)
	zb := fmt.Sprint(utf8.RuneCountInString(zero))
	if out, errOut, code := run("", "read", "--if-changed", "a.txt="+a, "--max-chars", zb, "a.txt:/zzz/"); code != 1 || out != zero || !strings.Contains(errOut, "mrw: incomplete: ") {
		t.Errorf("an exact budget before a missed note: exit %d, stdout %q, stderr %q", code, out, errOut)
	}
	if zj, _, _ := run("", "read", "--json", "--if-changed", "a.txt="+a, "--max-chars", zb, "a.txt:/zzz/"); !strings.Contains(zj, `"unchanged"`) {
		t.Errorf("an unchanged line that reached the caller is missing from the receipt: %s", zj)
	}

	// --json carries unchanged[{path, sha, lines, bytes}], and omits the key when empty.
	jout, _, code := run("", "read", "--json", "--if-changed", "a.txt="+a, "a.txt")
	var doc struct {
		Receipt struct {
			Unchanged []map[string]any `json:"unchanged"`
		} `json:"receipt"`
	}
	if err := json.Unmarshal([]byte(jout), &doc); err != nil || code != 0 {
		t.Fatalf("--json: exit %d, %v\n%s", code, err, jout)
	}
	want := map[string]any{"path": "a.txt", "sha": a, "lines": float64(3), "bytes": float64(20)}
	if len(doc.Receipt.Unchanged) != 1 || fmt.Sprint(doc.Receipt.Unchanged[0]) != fmt.Sprint(want) {
		t.Errorf("--json unchanged = %v, want [%v]", doc.Receipt.Unchanged, want)
	}
	jout, _, _ = run("", "read", "--json", "--if-changed", "a.txt=00000000", "a.txt")
	if strings.Contains(jout, `"unchanged"`) {
		t.Errorf("--json lists unchanged for a guard that did not match:\n%s", jout)
	}
}

// An unchanged answer licenses nothing at the CLI either: lines an earlier read
// did not serve stay unwritable, with a prior entry and with none.
func TestAnUnchangedAnswerAtTheCLILicensesNoWrite(t *testing.T) {
	run, _ := ifChangedRun(t, map[string]string{
		"a.txt": "alpha\nbravo\ncharlie\ndelta\n",
		"b.txt": "kilo\nlima\n",
	})
	sha := func(name string) string {
		out, _, _ := run("", "read", "--stat", name)
		return headerSha.FindStringSubmatch(out)[1]
	}
	a, b := sha("a.txt"), sha("b.txt")
	if _, _, code := run("", "read", "a.txt:1"); code != 0 { // serves lines 1-2
		t.Fatal("the first read failed")
	}
	for _, g := range []string{"a.txt=" + a, "b.txt=" + b} {
		if out, _, code := run("", "read", "--if-changed", g, strings.Split(g, "=")[0]); code != 0 || !strings.Contains(out, "unchanged since") {
			t.Fatalf("guard %s: exit %d, %q", g, code, out)
		}
	}
	for _, plan := range []string{"@@ a.txt 4 replace\nDELTA\n", "@@ b.txt 1 replace\nKILO\n"} {
		out, errOut, code := run(plan, "write", "--no-check", "-")
		if code != 1 || !strings.Contains(out+errOut, "has not been read") {
			t.Errorf("%q after an unchanged answer: exit %d\n%s%s", plan, code, out, errOut)
		}
	}
	// And the lines the first read did serve stay writable.
	if out, errOut, code := run("@@ a.txt 1 replace\nALPHA\n", "write", "--no-check", "-"); code != 0 {
		t.Errorf("a line the first read served was revoked: exit %d\n%s%s", code, out, errOut)
	}
}
