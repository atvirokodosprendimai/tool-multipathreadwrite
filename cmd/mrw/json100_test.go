package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter"
)

// refusalTrees returns a checkout whose declared check touches marker100, and
// a second one whose working set cannot be read: its file is a directory.
func refusalTrees(t *testing.T) (root, unreadable string) {
	t.Helper()
	files := map[string]string{
		".quality-harness.json": `{"check":"touch marker100"}` + "\n",
		"x.go":                  "package x\n",
	}
	root, unreadable = t.TempDir(), t.TempDir()
	writeFiles(t, root, files)
	writeFiles(t, unreadable, files)
	if out, code := runIn(t, unreadable, "iter", "add", "x.go"); code != 0 {
		t.Fatalf("iter add: exit %d:\n%s", code, out)
	}
	p, err := iter.ReadPath(unreadable)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(p, 0o755); err != nil {
		t.Fatal(err)
	}
	return root, unreadable
}

// oneDocument decodes s as exactly one JSON object and nothing after it: a
// second Decode must meet the end of the input, which More does not check.
func oneDocument(s string) (map[string]any, bool) {
	dec := json.NewDecoder(strings.NewReader(s))
	var doc map[string]any
	if err := dec.Decode(&doc); err != nil {
		return nil, false
	}
	var rest any
	if err := dec.Decode(&rest); err != io.EOF {
		return nil, false
	}
	return doc, true
}

// The review of #288: More reports another element of an enclosing array or
// object, not the end of the input, so trailing bytes passed. oneDocument
// accepts one object and refuses anything after it.
func TestOneDocumentRefusesAnythingAfterTheObject(t *testing.T) {
	if _, ok := oneDocument("{\"error\":\"x\"}\n"); !ok {
		t.Error("one object with a trailing newline was refused")
	}
	for _, s := range []string{`{"error":"x"}]`, `{"error":"x"}}`, `{"error":"x"}{"error":"y"}`, `{"error":"x"`, ``} {
		if _, ok := oneDocument(s); ok {
			t.Errorf("%q was accepted as exactly one document", s)
		}
	}
}

// ADR-100 T1. Under `mrw check --json` every refusal prints one document on
// stdout holding only the message the exit carries; without --json it prints
// nothing there. A consumer that parses stdout met an empty stream on these
// five and a document on the others, and must never read a verdict out of one.
func TestEveryCheckRefusalIsAJSONDocumentUnderJSON(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root, unreadable := refusalTrees(t)
	for _, c := range []struct {
		name string
		root string
		args []string
	}{
		{"edge whitespace", root, []string{"x.go "}},
		{"--full with a PATH", root, []string{"--full", "x.go"}},
		{"an unreadable working set", unreadable, nil},
		{"a path outside the root", root, []string{"../outside"}},
		{"a path not there", root, []string{"nosuchdir"}},
	} {
		stdout, err := runSplit(t, append([]string{"-C", c.root, "check", "--json"}, c.args...)...)
		if err == nil || exitCode(err) != exitUsage {
			t.Errorf("%s: err %v, want exit %d", c.name, err, exitUsage)
			continue
		}
		doc, ok := oneDocument(stdout)
		if e, _ := doc["error"].(string); !ok || len(doc) != 1 || e != err.Error() {
			t.Errorf("%s: stdout is not one document holding only the refusal %q:\n%s", c.name, err.Error(), stdout)
		}
		stdout, err = runSplit(t, append([]string{"-C", c.root, "check"}, c.args...)...)
		if err == nil || exitCode(err) != exitUsage || stdout != "" {
			t.Errorf("%s without --json: err %v, stdout %q; want exit %d and nothing on stdout", c.name, err, stdout, exitUsage)
		}
		if _, err := os.Stat(filepath.Join(c.root, "marker100")); err == nil {
			t.Fatalf("%s: the check ran; a refusal runs nothing", c.name)
		}
	}
	// The boundary: when no check could run, the Action prints its receipt and
	// then exits 2. That is still exactly one document, never a second one.
	none := t.TempDir()
	writeFiles(t, none, map[string]string{"a.txt": "x\n"})
	stdout, err := runSplit(t, "-C", none, "check", "--json", "--full")
	if doc, ok := oneDocument(stdout); err == nil || exitCode(err) != exitUsage || !ok || doc["ran"] != false {
		t.Errorf("no check could run: err %v, stdout not exactly one document with ran false:\n%s", err, stdout)
	}
}

// ADR-100 T2. At MRW_STEP_DEPTH 8 every form of mrw check hears ADR-095's depth
// refusal before an argument is judged or the working set is read; at 7 each
// form answers with its own refusal, as before.
func TestTheDepthLimitAnswersEveryCheckFormFirst(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root, unreadable := refusalTrees(t)
	t.Setenv("MRW_STEP_DEPTH", "8")
	for _, argv := range [][]string{
		{"-C", root, "check", "--full", "x.go"},
		{"-C", root, "check", "x.go "},
		{"-C", unreadable, "check"},
	} {
		if _, err := runSplit(t, argv...); err == nil || exitCode(err) != exitUsage || !strings.Contains(err.Error(), "MRW_STEP_DEPTH") {
			t.Errorf("%q at 8: err %v, want exit %d naming MRW_STEP_DEPTH", argv, err, exitUsage)
		}
	}
	stdout, err := runSplit(t, "-C", root, "check", "--json", "--full", "x.go")
	doc, ok := oneDocument(stdout)
	if e, _ := doc["error"].(string); err == nil || !ok || !strings.Contains(e, "MRW_STEP_DEPTH") {
		t.Errorf("check --json --full x.go at 8: err %v, stdout not one document naming MRW_STEP_DEPTH:\n%s", err, stdout)
	}
	t.Setenv("MRW_STEP_DEPTH", "7")
	if _, err := runSplit(t, "-C", root, "check", "--full", "x.go"); err == nil || err.Error() != "--full runs the whole project; it takes no PATH" {
		t.Errorf("check --full x.go at 7: err %v, want the --full refusal", err)
	}
}
