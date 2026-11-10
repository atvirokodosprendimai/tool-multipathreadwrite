package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// paddedTree holds x and "x " with different bytes, and skips where the
// filesystem cannot keep the two names apart.
func paddedTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for n, b := range map[string]string{"x": "plain\n", "x ": "padded\n"} {
		if err := os.WriteFile(filepath.Join(root, n), []byte(b), 0o644); err != nil {
			t.Skipf("this filesystem cannot hold %q: %v", n, err)
		}
	}
	if fi, err := os.ReadDir(root); err != nil || len(fi) != 2 {
		t.Skip("this filesystem folds names that differ by a trailing space")
	}
	return root
}

// plainTree holds x alone, for a test that needs no file named "x " — which
// Win32 cannot hold beside x (ADR-071 T4). paddedTree is for the tests that
// serve a file whose name ends in whitespace.
func plainTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "x"), []byte("plain\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// The tests below kept their ADR-069 names, which done tasks of that record
// lock; ADR-196 retargeted each at what urfave/cli v3.14.0 does with the same
// argv: the token reaches mrw as typed, so the padded name is the file served
// and its trimmed sibling never is (the precedent: ADR-187 kept ADR-054's).

// servesOnly fails unless out serves want's line and not plain's.
func servesOnly(t *testing.T, argv []string, out string, code int, want string) {
	t.Helper()
	if code != 0 || !strings.Contains(out, want) || strings.Contains(out, "| plain") {
		t.Errorf("%q: exit %d, want 0 serving %q and never x:\n%s", argv, code, want, out)
	}
}

// ADR-069 T5. A -- consumed as --grep's value is the pattern, and the padded
// path after it is read as typed: with only x on disk, x is never served.
func TestAFlagValueDoesNotEndThePaddedArgGuard(t *testing.T) {
	// x holds the pattern, so a trim of `x ` to x would serve it (the Codex
	// review of #628: with x holding no "--", a trim was invisible).
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"x": "has -- in it\n"})
	if out, code := runIn(t, root, "read", "--grep", "--", "x"); code != 0 || !strings.Contains(out, "| has -- in it") {
		t.Fatalf("read --grep -- x: exit %d, the pattern must match x:\n%s", code, out)
	}
	out, code := runIn(t, root, "read", "--grep", "--", "x ")
	if code == 0 || strings.Contains(out, "has -- in it") {
		t.Errorf("read --grep -- 'x ': exit %d, served x:\n%s", code, out)
	}
}

// ADR-069 T5. An attached flag value ending in a space is that value.
func TestAnAttachedFlagValueWithTrailingSpaceIsRefused(t *testing.T) {
	root := paddedTree(t)
	writeFiles(t, root, map[string]string{"l ": "x \n", "l": "x\n"})
	argv := []string{"read", "--files-from=l "}
	t.Chdir(root)
	out, code := runIn(t, root, argv...)
	servesOnly(t, argv, out, code, "| padded")
}

// ADR-069 T5. `iter add 'x '` adds the padded name, verb intact.
func TestTheIterRefusalKeepsTheVerb(t *testing.T) {
	root := paddedTree(t)
	if out, code := runIn(t, root, "iter", "add", "x "); code != 0 {
		t.Fatalf("iter add 'x ': exit %d:\n%s", code, out)
	}
	out, code := runIn(t, root, "read")
	servesOnly(t, []string{"read"}, out, code, "| padded")
}

// ADR-069 T6. A padded boolean flag name is the flag, and the padded path
// after it is read as typed.
func TestAPaddedBooleanFlagNameDoesNotHideAPaddedPath(t *testing.T) {
	argv := []string{"read", "--no-numbers ", "x "}
	out, code := runIn(t, paddedTree(t), argv...)
	if code != 0 || !strings.Contains(out, "\npadded\n") || strings.Contains(out, "plain") {
		t.Errorf("%q: exit %d, want the padded file unnumbered:\n%s", argv, code, out)
	}
}

// spacedRoots makes root/d and root/"d " each holding x, and reads x in "d ".
func spacedRoots(t *testing.T) (root, d, ds string) {
	t.Helper()
	root = t.TempDir()
	d, ds = filepath.Join(root, "d"), filepath.Join(root, "d ")
	for dir, b := range map[string]string{d: "plain\n", ds: "padded\n"} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Skip(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "x"), []byte(b), 0o644); err != nil {
			t.Skip(err)
		}
	}
	if entries, _ := os.ReadDir(root); len(entries) != 2 {
		t.Skip("this filesystem folds names that differ by a trailing space")
	}
	return root, d, ds
}

// ADR-069 T6. An attached root flag ending in a space names that directory,
// after a separate --root that took a -- as its value too.
func TestTheWholeArgvGuardReadsFlagValuesAndTheTerminator(t *testing.T) {
	root, _, ds := spacedRoots(t)
	argv := []string{"--root", "--", "--root=" + ds, "read", "x"}
	out, code := runIn(t, root, argv...)
	servesOnly(t, argv, out, code, "| padded")
}

// ADR-069 T6. An attached value ending in a newline or U+00A0 is that value.
func TestAnAttachedValueEndingInAnyWhitespaceIsRefused(t *testing.T) {
	root := paddedTree(t)
	t.Chdir(root)
	for _, name := range []string{"l\n", "l\u00a0"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x \n"), 0o644); err != nil {
			t.Skipf("this filesystem cannot hold %q: %v", name, err)
		}
		argv := []string{"read", "--files-from=" + name}
		out, code := runIn(t, root, argv...)
		servesOnly(t, argv, out, code, "| padded")
	}
}

// ADR-069 T7. An inherited --root after the verb, attached and padded, is
// the directory with the space: the write lands there, never in its sibling.
func TestAnInheritedRootFlagIsReadByBothGuards(t *testing.T) {
	root, d, ds := spacedRoots(t)
	if _, code := runIn(t, ds, "read", "x"); code != 0 {
		t.Fatal("read x in \"d \" failed")
	}
	plan := filepath.Join(t.TempDir(), "p.mrw")
	if err := os.WriteFile(plan, []byte("@@ x 1 replace\nnew\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, code := runIn(t, root, "write", "--root="+ds, plan); code != 0 {
		t.Fatalf("write --root='d ': exit %d:\n%s", code, out)
	}
	if b, _ := os.ReadFile(filepath.Join(ds, "x")); string(b) != "new\n" {
		t.Errorf("\"d \"/x holds %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(d, "x")); string(b) != "plain\n" {
		t.Errorf("d/x holds %q: the write reached the trimmed sibling", b)
	}
}

// ADR-069 T7. A single dash before a non-letter stops the parser, which
// keeps the token as given: ` -1= ` is served.
func TestASingleDashNonLetterTokenStopsTheParserAndTheGuard(t *testing.T) {
	root := plainTree(t)
	if err := os.WriteFile(filepath.Join(root, " -1= "), []byte("dashfile\n"), 0o644); err != nil {
		t.Skip(err)
	}
	t.Chdir(root)
	argv := []string{"read", " -1= "}
	out, code := runIn(t, root, argv...)
	if runtime.GOOS == "windows" {
		// NTFS stores " -1= " as " -1=", and ADR-071 refuses the spelling.
		if code == 0 || strings.Contains(out, "| dashfile") {
			t.Errorf("%q on Windows: exit %d, served the file under another name:\n%s", argv, code, out)
		}
		return
	}
	servesOnly(t, argv, out, code, "| dashfile")
}

// ADR-069 T8. A -- before the subcommand ends the root's options only: the
// subcommand's own padded --root after it is that directory.
func TestARootTerminatorDoesNotEndTheGuardBelowIt(t *testing.T) {
	root, _, ds := spacedRoots(t)
	argv := []string{"--", "read", "--json", "x"}
	if out, code := runIn(t, ds, argv...); code != 0 || !strings.Contains(out, "padded") {
		t.Errorf("%q in \"d \": exit %d:\n%s", argv, code, out)
	}
	argv = []string{"--", "stats", "--root=" + ds}
	if out, code := runIn(t, root, argv...); code != 0 {
		t.Errorf("%q: exit %d:\n%s", argv, code, out)
	}
}

// ADR-069 T8. A lone - no longer ends the parse: the flags after it are read.
func TestALoneDashEndsTheParseAndTheGuard(t *testing.T) {
	root := plainTree(t)
	withStdin(t, "@@ made.txt 0 create\nmade\n", func() {
		if out, code := runIn(t, root, "write", "-", "--dry-run"); code != 0 {
			t.Fatalf("write - --dry-run: exit %d:\n%s", code, out)
		}
	})
	if _, err := os.Stat(filepath.Join(root, "made.txt")); err == nil {
		t.Error("write - --dry-run wrote: the flag after - was dropped")
	}
}

// ADR-069 T9. A padded lone - is a plan file of that name, not stdin.
func TestAPaddedLoneDashIsRefusedBeforeTheGuardStops(t *testing.T) {
	root := plainTree(t)
	if err := os.WriteFile(filepath.Join(root, " - "), []byte("@@ c.txt 0 create\nfromfile\n"), 0o644); err != nil {
		t.Skip(err)
	}
	t.Chdir(root)
	withStdin(t, "@@ c.txt 0 create\nfromstdin\n", func() {
		if out, code := runIn(t, root, "write", " - "); code != 0 {
			t.Fatalf("write ' - ': exit %d:\n%s", code, out)
		}
	})
	if b, _ := os.ReadFile(filepath.Join(root, "c.txt")); string(b) != "fromfile\n" {
		t.Errorf("c.txt holds %q, want the plan from the file ' - '", b)
	}
}

// ADR-069 T10. A preserved stop token beside its trimmed twin: both served.
func TestAPreservedStopTokenIsNotJudgedAgainstAPositional(t *testing.T) {
	root := t.TempDir()
	for n, b := range map[string]string{" -1= ": "padded\n", "-1=": "twin\n"} {
		if err := os.WriteFile(filepath.Join(root, n), []byte(b), 0o644); err != nil {
			t.Skip(err)
		}
	}
	t.Chdir(root)
	out, code := runIn(t, root, "read", " -1= ", "-1=")
	if runtime.GOOS == "windows" {
		// NTFS stores " -1= " as " -1=": ADR-071 refuses that spelling, the twin is served.
		if code == 0 || strings.Contains(out, "| padded") || !strings.Contains(out, "| twin") {
			t.Errorf("on Windows: exit %d, want the padded spelling refused and the twin served:\n%s", code, out)
		}
		return
	}
	if code != 0 || !strings.Contains(out, "| padded") || !strings.Contains(out, "| twin") {
		t.Errorf("exit %d, want both served:\n%s", code, out)
	}
}

// ADR-069 T11. An iter verb is not a path: a padded verb is an unknown verb
// (v3.14.0 hands it over as typed), and a padded path after add is added.
func TestAnIterVerbIsNotJudgedAsAPath(t *testing.T) {
	root := paddedTree(t)
	if out, code := runIn(t, root, "iter", "add ", "x"); code != exitUsage || !strings.Contains(out, "unknown iter verb") {
		t.Errorf("iter 'add ' x: exit %d, want %d naming the verb:\n%s", code, exitUsage, out)
	}
	if out, code := runIn(t, root, "iter", "add", "x "); code != 0 {
		t.Errorf("iter add 'x ': exit %d:\n%s", code, out)
	}
}
