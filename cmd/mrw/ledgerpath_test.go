package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// ADR-068 T1, end to end through the CLI: after reading only "x " (or "x\r"),
// a write to "x" must be refused as unread and leave "x" alone, and a write to
// the file that WAS read must apply. Reproduced on v1.24.0 (2026-09-24) for
// "x ": the write to "x" applied, because the ledger loaded "x " as "x" and the
// two files held the same bytes, so the SHA guard could not tell them apart.
// The "\r" case came from the Codex review of #214 (2026-09-25): bufio.ScanLines
// drops a "\r" before the newline, so the ledger's scanner trimmed it.
//
// The read passes "--": urfave/cli trims a positional argument before it, so
// without "--" the read would reach "x" itself and prove nothing (BACKLOG).
//
// Windows strips a trailing space from a short path and refuses "\r" in a name,
// so both cases skip there; internal/seen covers the ledger on every platform.
func TestAReadOfATrailingSpacePathDoesNotLicenseItsTrimmedSibling(t *testing.T) {
	for _, name := range []string{"x ", "x\r"} {
		t.Run(strconv.Quote(name), func(t *testing.T) {
			root := t.TempDir()
			for _, n := range []string{"x", name} {
				if err := os.WriteFile(filepath.Join(root, n), []byte("same\n"), 0o644); err != nil {
					t.Skipf("this filesystem cannot hold both %q and %q: %v", "x", name, err)
				}
			}
			if fi, err := os.ReadDir(root); err != nil || len(fi) != 2 {
				t.Skipf("this filesystem folds %q and %q into one file", "x", name)
			}
			if _, err := readIn(t, root, "--", name); err != nil {
				t.Fatal(err)
			}

			out, code := writeIn(t, root, "--no-check", planFile(t, "@@ x 1 replace\nWROTE\n"))
			if code != 1 {
				t.Errorf("a write to the unread %q exited %d, want 1:\n%s", "x", code, out)
			}
			if b, _ := os.ReadFile(filepath.Join(root, "x")); string(b) != "same\n" {
				t.Errorf("a read of %q licensed a write to %q: it now holds %q", name, "x", b)
			}

			out, code = writeIn(t, root, "--no-check", planFile(t, "@@ \""+name+"\" 1 replace\nWROTE\n"))
			if code != 0 {
				t.Errorf("a write to the file that WAS read exited %d, want 0:\n%s", code, out)
			}
			if b, _ := os.ReadFile(filepath.Join(root, name)); string(b) != "WROTE\n" {
				t.Errorf("the write to %q did not land: %q", name, b)
			}
		})
	}
}
