//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func attributes(t *testing.T, path string) uint32 {
	t.Helper()
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	a, err := syscall.GetFileAttributes(p)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// ADR-076 T6. A write stages a new file and renames it over the old one, and
// the new file carried none of the old one's attributes: a Hidden file came out
// visible. Hidden and System are kept; a plain file gains neither.
func TestAWriteKeepsHiddenAndSystemAttributes(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"h.txt", "plain.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("a\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if out, err := exec.Command("attrib", "+h", "+s", filepath.Join(root, "h.txt")).CombinedOutput(); err != nil {
		t.Fatalf("attrib: %v\n%s", err, out)
	}
	const keep = syscall.FILE_ATTRIBUTE_HIDDEN | syscall.FILE_ATTRIBUTE_SYSTEM
	for name, want := range map[string]uint32{"h.txt": keep, "plain.txt": 0} {
		p := filepath.Join(t.TempDir(), name+".mrw")
		if err := os.WriteFile(p, []byte("@@ "+name+" 1 replace\nB\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if out, code := runIn(t, root, "write", "--no-check", "--force", p); code != 0 {
			t.Fatalf("write %s: exit %d:\n%s", name, code, out)
		}
		if got := attributes(t, filepath.Join(root, name)) & keep; got != want {
			t.Errorf("%s after a write: hidden/system = %#x, want %#x", name, got, want)
		}
	}
}

// ADR-076 T6. The read-only attribute is the read-only mark on Windows, and a
// write renamed over it at exit 0; it is refused, naming attrib -r.
func TestAReadOnlyFileIsRefusedOnWindows(t *testing.T) {
	root := t.TempDir()
	ro := filepath.Join(root, "ro.txt")
	if err := os.WriteFile(ro, []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("attrib", "+r", ro).CombinedOutput(); err != nil {
		t.Fatalf("attrib: %v\n%s", err, out)
	}
	t.Cleanup(func() { _ = exec.Command("attrib", "-r", ro).Run() })
	p := filepath.Join(t.TempDir(), "ro.mrw")
	if err := os.WriteFile(p, []byte("@@ ro.txt 1 replace\nB\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runIn(t, root, "write", "--no-check", "--force", p)
	if code != exitNotApplied || !strings.Contains(out, "read-only") || !strings.Contains(out, "attrib -r") {
		t.Errorf("write to a read-only file: exit %d, want %d naming attrib -r:\n%s", code, exitNotApplied, out)
	}
	if b, _ := os.ReadFile(ro); string(b) != "a\n" {
		t.Errorf("ro.txt changed: %q", b)
	}
}
