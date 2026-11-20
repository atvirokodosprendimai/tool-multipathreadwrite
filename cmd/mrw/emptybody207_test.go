package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-207. The write command refuses a delete whose body=@ file is empty before
// anything is written: exit 2, both files byte-identical, and under --json one
// document whose top-level error names the plan line and the file.
func TestTheCLIRefusesAnEmptyBodyFileOnADelete(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"f.txt": "a\nb\nc\n", "g.txt": "x\n", "empty.txt": ""})
	if out, code := runIn(t, root, "read", "f.txt", "g.txt"); code != 0 {
		t.Fatalf("read: exit %d:\n%s", code, out)
	}
	const doc = "@@ g.txt 1 replace\ny\n@@ f.txt 2 delete body=@empty.txt\n"
	snap := func() string {
		a, _ := os.ReadFile(filepath.Join(root, "f.txt"))
		b, _ := os.ReadFile(filepath.Join(root, "g.txt"))
		return string(a) + "|" + string(b)
	}
	before := snap()
	out, code := runIn(t, root, "write", "--no-check", planFile(t, doc))
	if code != exitUsage || !strings.Contains(out, "body=@empty.txt") || snap() != before {
		t.Errorf("exit %d, want %d naming body=@empty.txt, files untouched (%q vs %q):\n%s", code, exitUsage, snap(), before, out)
	}
	out, code = writeIn(t, root, "--no-check", "--json", planFile(t, doc))
	var r struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("--json is not one document: %v\n%s", err, out)
	}
	if code != exitUsage || !strings.Contains(r.Error, "line 3") || !strings.Contains(r.Error, "body=@empty.txt") || snap() != before {
		t.Errorf("--json: exit %d, error %q; want %d naming line 3 and body=@empty.txt, files untouched", code, r.Error, exitUsage)
	}
}
