package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-155. Through the command: a JSON path serves its value's lines and the
// line after them (ADR-161) and licenses exactly those — a write to the line
// after that is refused as unread, one to a line inside it applies — and a path
// that is not there exits 1 as an incomplete read.
func TestAJSONPathReadLicensesExactlyItsLines(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	doc := "{\n  \"name\": \"demo\",\n  \"scripts\": {\n    \"build\": \"go build\",\n    \"test\": \"go test\"\n  },\n  \"list\": [1]\n}\n"
	if err := os.WriteFile(filepath.Join(root, "f.json"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runIn(t, root, "read", "f.json:$.scripts")
	if code != 0 || !strings.Contains(out, "@@ 3-7") || strings.Contains(out, "demo") {
		t.Fatalf("read f.json:$.scripts exited %d, want lines 3-7 only:\n%s", code, out)
	}
	out, code = runIn(t, root, "write", "--no-check", planFile(t, "@@ f.json 8 replace\n}\n"))
	if code != 1 || !strings.Contains(out, "has not been read") {
		t.Errorf("a write to the line after the served lines exited %d, want 1 refused as unread:\n%s", code, out)
	}
	out, code = runIn(t, root, "write", "--no-check", planFile(t, "@@ f.json 4 replace\n    \"build\": \"go vet\",\n"))
	if code != 0 {
		t.Errorf("a write inside the span exited %d, want 0:\n%s", code, out)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "f.json")); !strings.Contains(string(b), "go vet") || !strings.Contains(string(b), "[1]") {
		t.Errorf("the file is not what the two writes should leave:\n%s", b)
	}
	out, code = runIn(t, root, "read", "f.json:$.nope")
	if code != 1 || !strings.Contains(out, `no name "nope"`) || !strings.Contains(out, "incomplete:") {
		t.Errorf("a missing path exited %d, want 1 naming it as incomplete:\n%s", code, out)
	}
}
