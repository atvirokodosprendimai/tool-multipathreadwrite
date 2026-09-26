package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-076 T4 through the CLI. The human receipt named neither the directories
// a plan made nor what a removed file had been — it printed `sha ` and an empty
// value. It names both now, and a rename says where the file went.
func TestTheReceiptNamesEveryPathTheWriteTouched(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	for name, body := range map[string]string{"gone.txt": "g\n", "mv.txt": "m\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := readIn(t, root, "gone.txt", "mv.txt"); err != nil {
		t.Fatal(err)
	}
	plan := planFile(t, "@@ n/deep/c.txt 0 create\nc\n@@ gone.txt - unlink\n@@ mv.txt - rename\nm2/mv.txt\n")
	out, code := writeIn(t, root, "--no-check", plan)
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	sep := string(filepath.Separator)
	for _, want := range []string{
		"created n" + sep + "\n",
		"created " + filepath.Join("n", "deep") + sep + "\n",
		"created m2" + sep + "\n",
		"removed gone.txt  1L -> 0L  was sha ",
		"removed mv.txt  1L -> 0L  was sha ",
		"renamed to " + filepath.Join("m2", "mv.txt"),
	} {
		if !strings.Contains(out, want) {
			t.Errorf("receipt lacks %q:\n%s", want, out)
		}
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasSuffix(line, "sha ") {
			t.Errorf("a receipt line ends in an empty sha: %q", line)
		}
	}
}

// ADR-076 T4: the human receipt of a write through an in-root symlink names the
// file that changed; a plain write names no target.
func TestTheReceiptNamesALinksTarget(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "real.txt"), []byte("r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := readIn(t, root, "link.txt"); err != nil {
		t.Fatal(err)
	}
	out, code := writeIn(t, root, "--no-check", planFile(t, "@@ link.txt 1 replace\nR\n"))
	if code != 0 || !strings.Contains(out, "wrote link.txt  1L -> 1L  sha ") || !strings.Contains(out, "target real.txt") {
		t.Fatalf("exit %d, the receipt does not name the link's target:\n%s", code, out)
	}
	if _, err := readIn(t, root, "real.txt"); err != nil {
		t.Fatal(err)
	}
	out, code = writeIn(t, root, "--no-check", planFile(t, "@@ real.txt 1 replace\nS\n"))
	if code != 0 || strings.Contains(out, "target") {
		t.Fatalf("exit %d, a plain write names a target:\n%s", code, out)
	}
}
