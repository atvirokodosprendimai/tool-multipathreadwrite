//go:build windows

package rooted

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"unsafe"
)

// The v1.61.0 Windows retest: a rename destination reaches GitDir spelled with
// the OS separator, and the refusal shows the path with "/" as every other does.
func TestADotGitRefusalSpellsThePathWithSlashes(t *testing.T) {
	err := GitDir(t.TempDir(), `.git\hooks\pre-push`)
	if err == nil || strings.Contains(err.Error(), `\`) || !strings.Contains(err.Error(), ".git/hooks/pre-push") {
		t.Errorf("GitDir with a backslash path = %v, want a refusal spelling it with slashes", err)
	}
}

// shortPath asks Windows for the 8.3 form of an existing path.
func shortPath(t *testing.T, p string) string {
	t.Helper()
	in, err := syscall.UTF16PtrFromString(p)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]uint16, 1024)
	n, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetShortPathNameW").Call(uintptr(unsafe.Pointer(in)), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 || int(n) > len(buf) {
		t.Fatalf("GetShortPathNameW(%q) = %d", p, n)
	}
	return syscall.UTF16ToString(buf[:n])
}

// ADR-143. The 8.3 name a real volume gives .git is refused, with the leaf not
// yet there as well as with it: the proof the rule needed beyond its predicate
// test. A volume with 8.3 names off has none to refuse.
func TestTheShortNameOfARealDotGitIsRefused(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "config"), []byte("[core]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	short := filepath.Base(shortPath(t, filepath.Join(root, ".git")))
	if strings.EqualFold(short, ".git") {
		t.Skip("this volume gives .git no 8.3 name")
	}
	// Measured 2026-10-10 on the CI windows runner: the name is GIT~1. The leaf
	// is checked there and not there, since a create has none yet.
	for _, p := range []string{short + "/config", short + "/hooks/pre-commit", strings.ToLower(short) + "/x"} {
		if err := GitDir(root, p); err == nil {
			t.Errorf("GitDir(%q) = nil, want a refusal: %s is .git", p, short)
		}
	}
	if err := GitDir(root, "git~12/config"); err != nil {
		t.Errorf("GitDir(git~12/config) = %v, want none: it is an ordinary name", err)
	}
}
