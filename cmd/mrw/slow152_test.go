package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
)

// ADR-152. The CLI receipt prints the check's hint on a line of its own, and
// the JSON receipt carries it as check.hint.
func TestTheCLIPrintsTheSlowCheckHint(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no sh: the check runs its command with sh -c")
	}
	root := t.TempDir()
	for name, body := range map[string]string{
		"a.txt":                 "one\n",
		".quality-harness.json": `{"check":"true"}`,
		"p.plan":                "@@ a.txt 1 replace\nuno\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	old := check.SlowAfter
	t.Cleanup(func() { check.SlowAfter = old })
	check.SlowAfter = 0

	if out, code := runIn(t, root, "read", "a.txt"); code != 0 {
		t.Fatalf("read exit %d: %s", code, out)
	}
	out, code := runIn(t, root, "write", "--check", filepath.Join(root, "p.plan"))
	if code != 0 {
		t.Fatalf("write exit %d: %s", code, out)
	}
	if n := strings.Count(out, "\nhint: "); n != 1 || !strings.Contains(out, `declare "scoped_check"`) {
		t.Errorf("want one hint line naming scoped_check, got %d:\n%s", n, out)
	}

	if err := os.WriteFile(filepath.Join(root, "p.plan"), []byte("@@ a.txt 1 replace\none\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code = runIn(t, root, "write", "--check", "--json", filepath.Join(root, "p.plan"))
	if code != 0 {
		t.Fatalf("write --json exit %d: %s", code, out)
	}
	var r struct {
		Check struct {
			Hint string `json:"hint"`
		} `json:"check"`
	}
	if err := json.Unmarshal([]byte(out[strings.Index(out, "{"):]), &r); err != nil {
		t.Fatalf("receipt: %v\n%s", err, out)
	}
	if !strings.Contains(r.Check.Hint, "scoped_check") {
		t.Errorf("the JSON receipt carries no check.hint: %s", out)
	}
}
