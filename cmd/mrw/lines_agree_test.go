package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeCR(t *testing.T, root, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fileBytes(t *testing.T, root, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// ADR-065 T1. A whole read of a CR-only file serves three lines, so a write to
// line 1 replaces exactly the line the caller was shown. It used to serve one
// line holding the whole file, and the same write replaced only "one".
func TestAWholeReadOfACROnlyFileLicensesOnlyTheLinesItServed(t *testing.T) {
	root := t.TempDir()
	writeCR(t, root, "cr.txt", "one\rtwo\rthree\r")
	out, err := readIn(t, root, "cr.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "    1| one\n") || !strings.Contains(out, "    3| three\n") {
		t.Fatalf("a whole read of a CR-only file did not serve its three lines:\n%q", out)
	}
	if _, code := writeIn(t, root, "--no-check", planFile(t, "@@ cr.txt 1 replace\nX\n")); code != 0 {
		t.Fatalf("a write to a served line was refused (exit %d)", code)
	}
	if got := fileBytes(t, root, "cr.txt"); got != "X\rtwo\rthree\r" {
		t.Fatalf("cr.txt = %q, want X\\rtwo\\rthree\\r", got)
	}
}

// ADR-065 T1 guard. A pure CRLF file read whole and then written applies —
// read hashes the raw bytes, so the ledger's sha agrees with the write's — and
// every terminator stays CRLF.
func TestACRLFFileReadWholeThenWrittenKeepsItsSha(t *testing.T) {
	root := t.TempDir()
	writeCR(t, root, "crlf.txt", "one\r\ntwo\r\nthree\r\n")
	if _, err := readIn(t, root, "crlf.txt"); err != nil {
		t.Fatal(err)
	}
	if out, code := writeIn(t, root, "--no-check", planFile(t, "@@ crlf.txt 2 replace\nTWO\n")); code != 0 {
		t.Fatalf("a write after a whole read was refused (exit %d):\n%s", code, out)
	}
	if got := fileBytes(t, root, "crlf.txt"); got != "one\r\nTWO\r\nthree\r\n" {
		t.Fatalf("crlf.txt = %q, want every line still CRLF", got)
	}
}

// ADR-065 T1 guard. A ranged read licenses only its span: after reading line
// 1 of a CR-only file, a write to line 2 is refused and the file is unchanged.
func TestAReadOfLineOneDoesNotLicenseLineTwoOfACROnlyFile(t *testing.T) {
	root := t.TempDir()
	writeCR(t, root, "cr.txt", "one\rtwo\rthree\r")
	if _, err := readIn(t, root, "cr.txt:1"); err != nil {
		t.Fatal(err)
	}
	if _, code := writeIn(t, root, "--no-check", planFile(t, "@@ cr.txt 2 replace\nX\n")); code != 1 {
		t.Fatalf("a write to an unserved line was not refused with exit 1 (exit %d)", code)
	}
	if got := fileBytes(t, root, "cr.txt"); got != "one\rtwo\rthree\r" {
		t.Fatalf("a refused write changed cr.txt: %q", got)
	}
}
