package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withStdin runs fn with os.Stdin reading content.
func withStdin(t *testing.T, content string, fn func()) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		_, _ = w.WriteString(content)
		_ = w.Close()
	}()
	old := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = old; _ = r.Close() }()
	fn()
}

// ADR-139. `write --create PATH` makes one file from standard input, through the
// same apply as a plan: the lines of the content, an existing file refused and
// left alone, and the flag refused beside a PLAN or a --format.
func TestWriteCreateMakesTheFileAndRefusesAnExistingOne(t *testing.T) {
	root := t.TempDir()
	content := "alpha\n@@ not.a.header 1 replace\nbeta"
	var out string
	var code int
	withStdin(t, content, func() { out, code = runIn(t, root, "write", "--no-check", "--create", "sub/new.txt") })
	if code != 0 {
		t.Fatalf("--create: exit %d\n%s", code, out)
	}
	b, err := os.ReadFile(filepath.Join(root, "sub", "new.txt"))
	if err != nil || string(b) != "alpha\n@@ not.a.header 1 replace\nbeta\n" {
		t.Errorf("the created file = %q (err %v), want the lines, each ending in a newline", b, err)
	}
	withStdin(t, "other\n", func() { out, code = runIn(t, root, "write", "--no-check", "--create", "sub/new.txt") })
	if code != 1 || !strings.Contains(out, "exists") {
		t.Errorf("--create of an existing file: exit %d, want 1 naming that it exists\n%s", code, out)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "sub", "new.txt")); !strings.HasPrefix(string(b), "alpha") {
		t.Errorf("the existing file was changed: %q", b)
	}
	plan := filepath.Join(t.TempDir(), "p.plan")
	if err := os.WriteFile(plan, []byte("@@ x.txt 0 create\nx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, argv := range [][]string{
		{"write", "--no-check", "--create", "a.txt", plan},
		{"write", "--no-check", "--create", "a.txt", "--format", "apply_patch"},
		{"write", "--no-check", "--create", ""},
		{"write", "--no-check", "--create", "/abs/a.txt"},
	} {
		withStdin(t, "x\n", func() { out, code = runIn(t, root, argv...) })
		if code != 2 {
			t.Errorf("mrw %s: exit %d, want 2\n%s", strings.Join(argv[1:], " "), code, out)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "a.txt")); err == nil {
		t.Error("a refused --create made a.txt")
	}
}
