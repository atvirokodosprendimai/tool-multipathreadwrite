package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-086 T4. encoding/json turns a byte that is not valid UTF-8 into U+FFFD,
// so once the plan header kept the byte, a --json receipt on a filesystem that
// holds it named a file other than the one written. A --json plan whose path or
// rename destination is not valid UTF-8 is refused before anything lands; the
// same plan without the invalid byte applies.
func TestAJSONPlanWithANameItCannotRepresentWritesNothing(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readIn(t, root, "b.txt"); err != nil {
		t.Fatal(err)
	}
	for _, doc := range []string{"@@ bad\xffname.txt 0 create\nx\n", "@@ b.txt - rename\nbad\xffname.txt\n"} {
		out, code := writeIn(t, root, "--no-check", "--json", planFile(t, doc))
		if code != 2 || !strings.Contains(out, "not valid UTF-8") {
			t.Errorf("--json plan %q: exit %d, want 2 naming the byte:\n%s", doc, code, out)
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Name() != "b.txt" {
			t.Errorf("--json plan %q changed the tree: %v", doc, entries)
		}
	}
	if out, code := writeIn(t, root, "--no-check", "--json", planFile(t, "@@ ok.txt 0 create\nx\n")); code != 0 {
		t.Errorf("a --json create with a valid name: exit %d, want 0\n%s", code, out)
	}
}

// ADR-086 T4 (Codex review of #254). The --json check saw a working-set
// pointer, not the name it expanded to, so `@@ @1 1 replace` under --json
// edited bad\xffname.txt while the receipt named another file. Only where the
// filesystem holds the byte (Linux CI); APFS refuses to create the name.
func TestAJSONPointerToANameItCannotRepresentWritesNothing(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "bad\xffname.txt"), []byte("x\n"), 0o644); err != nil {
		t.Skipf("this filesystem refuses the name (%v); no working set can point at it", err)
	}
	if _, err := readIn(t, root, "bad\xffname.txt"); err != nil {
		t.Fatal(err)
	}
	if out, code := runIn(t, root, "iter", "add", "bad\xffname.txt"); code != 0 {
		t.Fatalf("iter add: exit %d\n%s", code, out)
	}
	out, code := writeIn(t, root, "--no-check", "--json", planFile(t, "@@ @1 1 replace\nY\n"))
	if code != 2 || !strings.Contains(out, "not valid UTF-8") {
		t.Errorf("--json @1 to an invalid name: exit %d, want 2 naming the byte:\n%s", code, out)
	}
	if b, err := os.ReadFile(filepath.Join(root, "bad\xffname.txt")); err != nil || string(b) != "x\n" {
		t.Errorf("the pointed-at file changed: %q, %v", b, err)
	}
}
