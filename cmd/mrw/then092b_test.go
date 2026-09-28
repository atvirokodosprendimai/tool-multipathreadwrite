package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// ADR-092 T4 (Codex review of #266, finding 1). A step refusal under
// check --json is one document carrying error, as a write's is.
func TestACheckStepRefusalIsOneJSONDocument(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := stepsTree(t, "exit 0")
	broken := grepTree(t, map[string]string{".quality-harness.json": `{"check":"exit 0","steps":{"vet":""}}`})
	for _, c := range []struct {
		root string
		args []string
	}{
		{root, []string{"--then", "nope"}},
		{root, []string{"--then-sh", "  "}},
		{broken, []string{"--then", "vet"}},
	} {
		out, code := runIn(t, c.root, append([]string{"check", "--full", "--json"}, c.args...)...)
		if code != exitUsage {
			t.Fatalf("%v: exit %d, want %d:\n%s", c.args, code, exitUsage, out)
		}
		var doc struct {
			Error string `json:"error"`
		}
		end := strings.LastIndex(out, "}")
		if end < 0 || json.Unmarshal([]byte(out[:end+1]), &doc) != nil || doc.Error == "" {
			t.Errorf("%v: no JSON refusal document:\n%s", c.args, out)
		}
	}
}

// ADR-092 T4 (finding 2). With no temp directory the check cannot create its
// log and returns an error: the write has landed, and the steps it asked for
// are still named, not_run, in both receipts.
func TestAWriteWhoseCheckCannotRunStillNamesItsSteps(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := stepsTree(t, "exit 0")
	plan := primed(t, root)
	gone := filepath.Join(t.TempDir(), "gone")
	t.Setenv("TMPDIR", gone)
	if runtime.GOOS == "windows" {
		t.Setenv("TMP", gone)
		t.Setenv("TEMP", gone)
	}
	out, code := writeIn(t, root, "--json", "--then", "a", "--then-sh", "echo x >> log", plan)
	if code != exitUsage {
		t.Fatalf("exit %d, want %d:\n%s", code, exitUsage, out)
	}
	r := parseThen(t, out)
	if got := thenStatuses(r); !reflect.DeepEqual(got, []string{"not_run", "not_run"}) || r.Error == "" {
		t.Errorf("statuses %v, error %q, want both not_run beside the error:\n%s", got, r.Error, out)
	}
	plan2 := planFile(t, "@@ a.go 2 replace anchor=\"func A\"\nfunc A() { _ = 2 }\n")
	out, code = writeIn(t, root, "--then", "a", plan2)
	if code != exitUsage || !strings.Contains(out, "NOT RUN") {
		t.Errorf("the human receipt does not name the step not run: exit %d:\n%s", code, out)
	}
	if got := logOf(t, root); got != "" {
		t.Errorf("a step ran: %q", got)
	}
}

// ADR-092 T4 (finding 3). A passing step whose output ran past tail_lines keeps
// its log (ADR-080), and the receipt says where.
func TestAPassingStepThatKeptItsLogNamesIt(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\n",
		".quality-harness.json": `{"check":"exit 0","tail_lines":2,"steps":{"loud":"seq 1 50"}}`,
	})
	out, code := writeIn(t, root, "--no-check", "--then", "loud", primed(t, root))
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	i := strings.Index(out, "earlier line(s) in ")
	if i < 0 {
		t.Fatalf("the receipt does not name the kept log:\n%s", out)
	}
	log := strings.TrimSpace(strings.SplitN(out[i+len("earlier line(s) in "):], "\n", 2)[0])
	if _, err := os.Stat(log); err != nil {
		t.Errorf("the named log %q: %v", log, err)
	}
	_ = os.Remove(log)
}
