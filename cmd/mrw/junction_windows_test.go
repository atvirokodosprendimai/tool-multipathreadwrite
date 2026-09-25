//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-071 T1, end to end. The Windows round read, replaced, created, renamed
// and unlinked through a junction inside --root, and every one landed outside
// it at exit 0. --force bypasses the read ledger so each refusal below is the
// root's, not the ledger's; the outside directory must be byte-identical after.
func TestAJunctionCannotCarryAnyOpOutOfTheRoot(t *testing.T) {
	base := t.TempDir()
	root, outside := filepath.Join(base, "root"), filepath.Join(base, "outside")
	for p, b := range map[string]string{
		filepath.Join(outside, "secret.txt"): "secret\n",
		filepath.Join(root, "a.txt"):         "a\n",
	} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(b), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", filepath.Join(root, "j"), outside).CombinedOutput(); err != nil {
		t.Fatalf("mklink /J: %v\n%s", err, out)
	}
	listing := func() string {
		es, err := os.ReadDir(outside)
		if err != nil {
			t.Fatal(err)
		}
		var b strings.Builder
		for _, e := range es {
			c, _ := os.ReadFile(filepath.Join(outside, e.Name()))
			b.WriteString(e.Name() + "=" + string(c) + ";")
		}
		return b.String()
	}
	before := listing()

	if out, code := runIn(t, root, "read", `j\secret.txt`); code == 0 || strings.Contains(out, "| secret") {
		t.Errorf("read through a junction out of the root served it, exit %d:\n%s", code, out)
	}
	plans := t.TempDir()
	for name, plan := range map[string]string{
		"replace": "@@ j\\secret.txt 1 replace\nX\n",
		"create":  "@@ j\\new.txt - create\nnew\n",
		"rename":  "@@ a.txt - rename\nj\\moved.txt\n",
		"unlink":  "@@ j\\secret.txt - unlink\n",
	} {
		p := filepath.Join(plans, name+".mrw")
		if err := os.WriteFile(p, []byte(plan), 0o644); err != nil {
			t.Fatal(err)
		}
		out, code := runIn(t, root, "write", "--no-check", "--force", p)
		if code != exitNotApplied || !strings.Contains(out, "outside the root") {
			t.Errorf("%s through a junction out of the root: exit %d, want %d refusing it as outside the root:\n%s", name, code, exitNotApplied, out)
		}
	}
	if after := listing(); after != before {
		t.Errorf("the directory outside the root changed:\nbefore %s\nafter  %s", before, after)
	}
	if _, err := os.Stat(filepath.Join(root, "a.txt")); err != nil {
		t.Errorf("a.txt is gone from the root: %v", err)
	}
}

// ADR-071, review of #228 (S1). A root reached through a junction must still
// serve an absolute path inside it: read's pre-screen compared the argument,
// resolved only by EvalSymlinks (which stops at a junction), with a root that
// Abs had followed through it, so the two were spelled differently.
func TestAnAbsolutePathUnderAJunctionedRootIsServed(t *testing.T) {
	base := t.TempDir()
	root, alias := filepath.Join(base, "root"), filepath.Join(base, "alias")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("inside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", alias, root).CombinedOutput(); err != nil {
		t.Fatalf("mklink /J: %v\n%s", err, out)
	}
	abs := filepath.Join(alias, "a.txt")
	if out, code := runIn(t, alias, "read", abs); code != 0 || !strings.Contains(out, "| inside") {
		t.Errorf("read %s under a root reached through a junction: exit %d:\n%s", abs, code, out)
	}
	if out, code := runIn(t, alias, "read", "--grep", "inside", abs); code != 0 || !strings.Contains(out, "| inside") {
		t.Errorf("read --grep inside %s under a junctioned root: exit %d:\n%s", abs, code, out)
	}
}
