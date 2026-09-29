package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
)

// placeholderTree holds a Go file and a harness whose check touches checked
// and whose one step, fmt, appends to log with arg in its command.
func placeholderTree(t *testing.T, arg string) string {
	t.Helper()
	return grepTree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\n",
		".quality-harness.json": "{\"check\":\"touch checked\",\"steps\":{\"fmt\":\"echo fmt " + arg + " >> log\"}}",
	})
}

// jsonDoc runs mrw -C root with argv and decodes its stdout alone — not stderr,
// not the returned error — as exactly one JSON document: none, a second one, or
// anything but space after the first fails the test.
func jsonDoc(t *testing.T, root string, argv ...string) (thenReceipt, int, string) {
	t.Helper()
	stdout, err := runSplit(t, append([]string{"-C", root}, argv...)...)
	code := 0
	if err != nil {
		code = exitCode(err)
	}
	dec := json.NewDecoder(strings.NewReader(stdout))
	var r thenReceipt
	if err := dec.Decode(&r); err != nil {
		t.Fatalf("no JSON document on stdout: %v\n%s", err, stdout)
	}
	if err := dec.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
		t.Fatalf("stdout holds more than one JSON document (%v):\n%s", err, stdout)
	}
	return r, code, stdout
}

// ADR-094 T1, the record's Enforced-by. On write, a declared step and a
// --then-sh holding {files} or {packages} are each refused, exit 2, before
// anything is written or run, naming the step and the token; under --json each
// is one refusal document. The declared refusal comes after the plan parsed and
// is one refused_apply; the ad-hoc one comes before the plan is read — a plan
// path that does not exist is never reached — and is not counted (ADR-083).
func TestAStepWithAPlaceholderIsRefusedBeforeAnythingIsWritten(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := placeholderTree(t, "{files}")
	plan := primed(t, root)
	before, _ := os.ReadFile(filepath.Join(root, "a.go"))
	tally := func(want int) {
		t.Helper()
		got, err := authoring.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		if got["refused_apply"] != want || got.Plans() != want {
			t.Errorf("tally %v, want refused_apply %d and nothing else", got, want)
		}
	}
	untouched := func(what, out string) {
		t.Helper()
		if after, _ := os.ReadFile(filepath.Join(root, "a.go")); string(after) != string(before) {
			t.Errorf("%s: a.go changed:\n%s", what, out)
		}
		if got := logOf(t, root); got != "" {
			t.Errorf("%s: a step ran: %q", what, got)
		}
	}

	out, code := runIn(t, root, "write", "--then", "fmt", plan)
	if code != exitUsage || !strings.Contains(out, "step \"fmt\"") || !strings.Contains(out, "{files}") {
		t.Errorf("declared: exit %d, want 2 naming step \"fmt\" and {files}:\n%s", code, out)
	}
	untouched("declared", out)
	tally(1)
	r, code, out := jsonDoc(t, root, "write", "--json", "--then", "fmt", plan)
	if code != exitUsage || !strings.Contains(r.Error, "{files}") {
		t.Errorf("declared --json: exit %d, error %q:\n%s", code, r.Error, out)
	}
	untouched("declared --json", out)
	tally(2)

	adhoc := "echo x {packages} >> log"
	out, code = runIn(t, root, "write", "--then-sh", adhoc, plan)
	if code != exitUsage || !strings.Contains(out, "--then-sh") || !strings.Contains(out, "{packages}") {
		t.Errorf("ad hoc: exit %d, want 2 naming --then-sh and {packages}:\n%s", code, out)
	}
	untouched("ad hoc", out)
	r, code, out = jsonDoc(t, root, "write", "--json", "--then-sh", adhoc, plan)
	if code != exitUsage || !strings.Contains(r.Error, "{packages}") {
		t.Errorf("ad hoc --json: exit %d, error %q:\n%s", code, r.Error, out)
	}
	untouched("ad hoc --json", out)
	missing := filepath.Join(t.TempDir(), "missing.mrw")
	out, code = runIn(t, root, "write", "--then-sh", adhoc, missing)
	if code != exitUsage || !strings.Contains(out, "{packages}") || strings.Contains(out, "missing.mrw") {
		t.Errorf("ad hoc, a plan that does not exist: exit %d, want 2 naming the token and not the path:\n%s", code, out)
	}
	tally(2)

	t.Run("the same steps with a path in place of the token run", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := placeholderTree(t, "a.go")
		out, code := runIn(t, root, "write", "--then", "fmt", "--then-sh", "echo x a.go >> log", primed(t, root))
		if code != 0 || logOf(t, root) != "fmt a.go\nx a.go\n" {
			t.Errorf("exit %d, log %q, want 0 and both markers:\n%s", code, logOf(t, root), out)
		}
	})
}

// ADR-094 T1. On check, the same two refusals land before the project's check
// runs: the check touches checked, so a refusal moved below check.Run would
// leave it. check counts nothing in the tally.
func TestCheckRunsNoStepThatHoldsAPlaceholder(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := placeholderTree(t, "{files}")
	ran := func(name string) bool {
		_, err := os.Stat(filepath.Join(root, name))
		return err == nil
	}
	for _, c := range []struct {
		args []string
		tok  string
	}{
		{[]string{"--then", "fmt"}, "{files}"},
		{[]string{"--then-sh", "echo x {packages} >> log"}, "{packages}"},
	} {
		for _, asJSON := range []bool{false, true} {
			argv := append([]string{"check"}, c.args...)
			var out string
			var code int
			if asJSON {
				var r thenReceipt
				r, code, out = jsonDoc(t, root, append(argv, "--json", "a.go")...)
				if !strings.Contains(r.Error, c.tok) {
					t.Errorf("%v --json: the refusal document carries %q", c.args, r.Error)
				}
			} else {
				out, code = runIn(t, root, append(argv, "a.go")...)
			}
			if code != exitUsage || !strings.Contains(out, c.tok) {
				t.Errorf("%v json %v: exit %d, want 2 naming %s:\n%s", c.args, asJSON, code, c.tok, out)
			}
			if ran("checked") || ran("log") {
				t.Errorf("%v json %v: the check ran %v, a step ran %v:\n%s", c.args, asJSON, ran("checked"), ran("log"), out)
			}
		}
	}
	if got, err := authoring.Load(root); err != nil || got.Plans() != 0 {
		t.Errorf("check counted a plan: %v %v", got, err)
	}

	t.Run("the token-free siblings run after a passing check", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := placeholderTree(t, "a.go")
		out, code := runIn(t, root, "check", "--then", "fmt", "--then-sh", "echo x a.go >> log", "a.go")
		if _, err := os.Stat(filepath.Join(root, "checked")); code != 0 || err != nil || logOf(t, root) != "fmt a.go\nx a.go\n" {
			t.Errorf("exit %d, checked %v, log %q:\n%s", code, err, logOf(t, root), out)
		}
	})
}
