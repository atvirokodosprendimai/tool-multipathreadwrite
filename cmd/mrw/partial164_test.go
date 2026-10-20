package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-164 Decision 4 in the CLI. A write, or `mrw check PATH…`, that places
// some of the files runs those and prints the rest on a line of its own; the
// JSON receipt carries them as check.outside_scope.
func TestTheCLIPrintsTheFilesOutsideTheScopedCheck(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no sh: the check runs its command with sh -c")
	}
	root := t.TempDir()
	for name, body := range map[string]string{
		"a.go":                  "package a\n",
		"run.sh":                "true\n",
		".quality-harness.json": `{"check":"echo WHOLE","scoped_check":"echo {packages}"}`,
		"p.plan":                "@@ a.go 1 replace\npackage a // x\n@@ run.sh 1 replace\nfalse\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	line := `outside the scoped check: run.sh — scoped_check "echo {packages}" runs {packages}, which names Go packages only`

	if out, code := runIn(t, root, "read", "a.go", "run.sh"); code != 0 {
		t.Fatalf("read exit %d: %s", code, out)
	}
	out, code := runIn(t, root, "write", "--check", "--json", filepath.Join(root, "p.plan"))
	if code != 0 {
		t.Fatalf("write --json exit %d: %s", code, out)
	}
	var r struct {
		Check struct {
			Command      string   `json:"command"`
			OutsideScope []string `json:"outside_scope"`
		} `json:"check"`
	}
	if err := json.Unmarshal([]byte(out[strings.Index(out, "{"):]), &r); err != nil {
		t.Fatalf("receipt: %v\n%s", err, out)
	}
	if r.Check.Command != "echo ." || len(r.Check.OutsideScope) != 1 || r.Check.OutsideScope[0] != "run.sh" {
		t.Errorf("the JSON receipt: command %q outside %v", r.Check.Command, r.Check.OutsideScope)
	}

	// mrw check PATH… follows the same rule, and prints the line once.
	out, code = runIn(t, root, "check", "a.go", "run.sh")
	if code != 0 {
		t.Fatalf("check exit %d: %s", code, out)
	}
	if n := strings.Count(out, line+"\n"); n != 1 {
		t.Errorf("want the outside line once, got %d:\n%s", n, out)
	}

	// The pair: nothing outside, no line; nothing placed, the whole check.
	out, code = runIn(t, root, "check", "a.go")
	if code != 0 || strings.Contains(out, "outside the scoped check") {
		t.Errorf("a fully placed check, exit %d:\n%s", code, out)
	}
	out, code = runIn(t, root, "check", "run.sh")
	if code != 0 || !strings.Contains(out, "echo WHOLE") || strings.Contains(out, "outside the scoped check") {
		t.Errorf("a check placing nothing, exit %d:\n%s", code, out)
	}
}
