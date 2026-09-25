//go:build windows

package rooted

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// mklinkJ makes a junction the way a Windows user does. A junction needs no
// privilege, so unlike a symlink this never skips.
func mklinkJ(t *testing.T, link, target string) {
	t.Helper()
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
		t.Fatalf("mklink /J %s %s: %v\n%s", link, target, err, out)
	}
}

func mustWrite(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// ADR-071 T1, the escape three Windows sessions reproduced: Go 1.23+ reports a
// junction as irregular and EvalSymlinks no longer follows it, so Resolve
// judged j\secret.txt by its spelling. A junction that stays inside the root,
// and a root reached through a junction, must still resolve.
func TestResolveRefusesAJunctionOutOfTheRoot(t *testing.T) {
	base := t.TempDir()
	root, outside := filepath.Join(base, "root"), filepath.Join(base, "outside")
	mustWrite(t, filepath.Join(outside, "secret.txt"), "secret\n")
	mustWrite(t, filepath.Join(root, "inner", "ok.txt"), "ok\n")
	mklinkJ(t, filepath.Join(root, "j"), outside)
	mklinkJ(t, filepath.Join(root, "in"), filepath.Join(root, "inner"))

	for _, p := range []string{"j", `j\secret.txt`, `j\new.txt`, `j\deeper\new.txt`} {
		if got, err := Resolve(root, p); err == nil {
			t.Errorf("Resolve(%s) = %s: a junction out of the root was followed lexically", p, got)
		}
	}
	if _, err := Resolve(root, `in\ok.txt`); err != nil {
		t.Errorf("a junction that stays inside the root was refused: %v", err)
	}
	alias := filepath.Join(base, "alias")
	mklinkJ(t, alias, root)
	if _, err := Resolve(alias, `inner\ok.txt`); err != nil {
		t.Errorf("a root reached through a junction refused its own file: %v", err)
	}
}

// ADR-071 T3: Win32 maps b.txt. and "b.txt " to b.txt and b.txt::$DATA to its
// default stream, so each is refused by name; the real name still resolves,
// and a root spelled with a trailing space or dot is refused too.
func TestResolveRefusesAWin32Alias(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "b.txt"), "b\n")
	for _, p := range []string{"b.txt.", "b.txt ", `b.txt::$DATA`, `sub.\b.txt`} {
		if got, err := Resolve(root, p); err == nil {
			t.Errorf("Resolve(%q) = %s: a Win32 alias was accepted", p, got)
		}
	}
	if _, err := Resolve(root, "b.txt"); err != nil {
		t.Errorf("the real name was refused: %v", err)
	}
	for _, r := range []string{root + " ", root + "."} {
		if _, err := Abs(r); err == nil {
			t.Errorf("Abs(%q) accepted a root Windows would read as %s", r, root)
		}
	}
}
