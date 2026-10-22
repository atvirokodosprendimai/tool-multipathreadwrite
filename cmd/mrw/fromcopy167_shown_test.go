package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Zy, 2026-10-22 (ADR-167): a from= source line must have been SHOWN. The three
// routes that license lines without showing them are refused, exit 1, nothing
// written; the pair is a plain read of the same lines, which copies.
func TestAFromCopyRefusesLinesLicensedButNeverShown(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	const src = "// one\n// two\n// three\n// four\n// five\n// six\n"
	const plan = "@@ b.txt 1 insert-after from=a.go:3-5\n"
	mkTree := func(t *testing.T) string {
		return grepTree(t, map[string]string{"a.go": src, "b.txt": "x\ny\n"})
	}
	slurp := func(t *testing.T, root, name string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	refused := func(t *testing.T, root, route string) {
		t.Helper()
		out, code := writeIn(t, root, "--no-check", planFile(t, plan))
		if code != 1 || slurp(t, root, "b.txt") != "x\ny\n" {
			t.Errorf("%s: exit %d, want 1 and nothing written:\n%s", route, code, out)
		}
		if strings.Contains(out, "// three") {
			t.Errorf("%s: the refusal discloses a source line:\n%s", route, out)
		}
	}

	t.Run("a --quiet read licenses the source and shows nothing", func(t *testing.T) {
		root := mkTree(t)
		if out, err := readIn(t, root, "--quiet", "a.go:3-5"); err != nil {
			t.Fatalf("quiet read: %v\n%s", err, out)
		}
		if out, err := readIn(t, root, "b.txt"); err != nil {
			t.Fatalf("read: %v\n%s", err, out)
		}
		refused(t, root, "--quiet")
	})

	t.Run("a --force write to an unread file shows it nothing", func(t *testing.T) {
		root := mkTree(t)
		if out, code := writeIn(t, root, "--no-check", "--force", planFile(t, "@@ a.go 1 replace\n// ONE\n")); code != 0 {
			t.Fatalf("forced write: exit %d\n%s", code, out)
		}
		if out, err := readIn(t, root, "b.txt"); err != nil {
			t.Fatalf("read: %v\n%s", err, out)
		}
		refused(t, root, "--force")
	})

	t.Run("a --sub write licenses its lines and shows none", func(t *testing.T) {
		root := mkTree(t)
		if out, code := writeIn(t, root, "--no-check", "--grep", "// (three|four|five)", "--sub", "// $1!", "a.go"); code != 0 {
			t.Fatalf("--sub write: exit %d\n%s", code, out)
		}
		if out, err := readIn(t, root, "b.txt"); err != nil {
			t.Fatalf("read: %v\n%s", err, out)
		}
		refused(t, root, "--sub")
	})

	t.Run("the pair: a plain read of the same lines copies", func(t *testing.T) {
		root := mkTree(t)
		if out, err := readIn(t, root, "a.go:3-5", "b.txt"); err != nil {
			t.Fatalf("read: %v\n%s", err, out)
		}
		if out, code := writeIn(t, root, "--no-check", planFile(t, plan)); code != 0 {
			t.Fatalf("exit %d:\n%s", code, out)
		}
	})
}

// ADR-157 Decision 2 for a from= refusal: under a newer ledger the sentence is
// said once, in the FAIL line, which names the source.
func TestANewerLedgersFromCopyRefusalSaysItOnce(t *testing.T) {
	root := newerRoot(t)
	out, errOut, code := writeUnderNewer(t, root, "@@ g.txt 1 insert-after from=f.txt:1-2\n")
	fails := failLines(out)
	if code != 1 || len(fails) != 1 || !strings.Contains(fails[0], "f.txt cannot be copied by this mrw: the read ledger was "+newerKey) {
		t.Fatalf("exit %d, stdout %q stderr %q", code, out, errOut)
	}
	if n := strings.Count(out+errOut, newerKey); n != 1 {
		t.Errorf("the sentence appears %d times, want once: stdout %q stderr %q", n, out, errOut)
	}
}
