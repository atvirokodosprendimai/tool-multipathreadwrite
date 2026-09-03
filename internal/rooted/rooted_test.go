package rooted

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// realRoot is the root as Resolve sees it. On macOS t.TempDir() hands back a
// path under /var, which is itself a symlink to /private/var — so a test that
// compares against the unevaluated string is asserting the platform, not the
// boundary.
func realRoot(t *testing.T, root string) string {
	t.Helper()
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	return real
}

func TestResolveAcceptsWhatIsInside(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"a.go", "sub/a.go", "sub/../sub/a.go", "./a.go", "."} {
		got, err := Resolve(root, path)
		if err != nil {
			t.Errorf("Resolve(%q) refused a path inside the root: %v", path, err)
			continue
		}
		if !strings.HasPrefix(got, realRoot(t, root)) {
			t.Errorf("Resolve(%q) = %q, which is not under %q", path, got, realRoot(t, root))
		}
	}
}

func TestResolveRefusesWhatIsOutside(t *testing.T) {
	root := t.TempDir()

	for _, path := range []string{"../escape.txt", "sub/../../escape.txt", "/etc/hosts"} {
		if _, err := Resolve(root, path); err == nil && path != "/etc/hosts" {
			t.Errorf("Resolve(%q) was accepted", path)
		}
	}
	// An absolute path is joined onto the root rather than honoured, so it
	// lands inside; the check is that it cannot ESCAPE, not that it is
	// rejected. Stated here because the opposite is easy to assume.
	if got, err := Resolve(root, "/etc/hosts"); err != nil || !strings.HasPrefix(got, realRoot(t, root)) {
		t.Errorf("an absolute path resolved to %q (err %v); want it joined under the root", got, err)
	}
}

// A symlink is the other way out, and the reason the check resolves before it
// compares: /etc/hosts was read through one.
func TestResolveRefusesASymlinkOutOfTheRoot(t *testing.T) {
	outer := t.TempDir()
	root := filepath.Join(outer, "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(outer, "secret.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := Resolve(root, "link.txt"); err == nil {
		t.Error("a symlink pointing out of the root was accepted")
	}
	if _, err := Resolve(root, "nothere.txt"); err != nil {
		t.Errorf("a file that does not exist yet was refused: %v — create needs it", err)
	}
}

// The separator in the prefix check is load-bearing: without it a sibling whose
// name merely STARTS with the root's counts as inside.
func TestASiblingWithASharedPrefixIsOutside(t *testing.T) {
	// Built with the platform's own separator. The hardcoded POSIX literals
	// this test used to carry made it assert the opposite of its name on
	// Windows: "/repo/a.go" has no `\repo\` prefix, so a real child read as
	// outside and the test failed for a reason that was not the boundary's.
	root := filepath.Join(string(filepath.Separator), "repo")
	sibling := root + "-backup" + string(filepath.Separator) + "secret"
	if Contains(root, sibling) {
		t.Errorf("%q was judged to be inside %q", sibling, root)
	}
	if !Contains(root, filepath.Join(root, "a.go")) || !Contains(root, root) {
		t.Error("a real child, or the root itself, was judged to be outside")
	}
}

// IsRooted is what every caller asks before deciding whether a path is theirs
// to resolve. The case it exists for cannot be REACHED on POSIX — a backslash
// is an ordinary filename character there — so half this table is asserted
// against the running platform rather than against a constant.
func TestIsRootedAnswersForTheRunningPlatform(t *testing.T) {
	if !IsRooted("/etc/hosts") {
		t.Error(`IsRooted("/etc/hosts") = false; a slash-rooted path is never root-relative, on any platform`)
	}
	for _, p := range []string{"", "a/b", "a", filepath.Join("sub", "f.go")} {
		if IsRooted(p) {
			t.Errorf("IsRooted(%q) = true; that is an ordinary relative path", p)
		}
	}
	windows := filepath.Separator == '\\'
	for _, p := range []string{`\etc\hosts`, `C:\etc`, `C:etc`} {
		if got := IsRooted(p); got != windows {
			t.Errorf("IsRooted(%q) = %v here; want %v — a leading backslash or a drive "+
				"letter names a location on Windows and is an ordinary relative path elsewhere", p, got, windows)
		}
	}
}

// The root is exempt: Resolve already canonicalises it, so a checkout reached
// through a symlink is usable and its entries are compared against the real
// path. Without this, a walk of /tmp-on-macOS would refuse its own root.
func TestASymlinkedRootIsStillUsable(t *testing.T) {
	outer := t.TempDir()
	real := filepath.Join(outer, "real-root")
	if err := os.MkdirAll(filepath.Join(real, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(outer, "link-root")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	// Through Resolve, because Resolve is what canonicalises — an earlier
	// version of this compared a raw t.TempDir() path against an EvalSymlinks'd
	// root and failed on macOS, where /var is a symlink to /private/var. The
	// point of the test is the ROOT being symlinked, not the caller doing the
	// resolving by hand.
	if _, err := Resolve(link, "sub"); err != nil {
		t.Errorf("a subdirectory of a symlinked root was refused: %v", err)
	}
}
