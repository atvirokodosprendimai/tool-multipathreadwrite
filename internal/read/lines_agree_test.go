package read

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func writeFile(t *testing.T, root, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// ADR-065 T1. A CR-only file is served as the lines the write engine
// addresses (ADR-005 §3). It used to be served as ONE line holding every
// `\r`, so a write to its line 2 applied though read had never served it.
func TestACROnlyFileIsServedAsTheLinesAWriteAddresses(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "cr.txt", "one\rtwo\rthree\r")
	out, _ := run(t, root, Options{Numbers: true}, "cr.txt")
	if !strings.Contains(out, "==> cr.txt  3L") {
		t.Fatalf("a CR-only file of three lines is not served as 3L:\n%q", out)
	}
	out, problems := run(t, root, Options{Numbers: true}, "cr.txt:2")
	if problems != 0 || !strings.Contains(out, "    2| two\n") || strings.Contains(out, "one") {
		t.Fatalf("cr.txt:2 does not serve exactly the line two (problems=%d):\n%q", problems, out)
	}
}

// ADR-065 T1. A CRLF line is served without its terminator, so a read's
// `/one$/` matches where the write's does (ADR-036).
func TestACRLFLineIsServedWithoutItsTerminator(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "crlf.txt", "one\r\ntwo\r\nthree\r\n")
	out, problems := run(t, root, Options{Numbers: true}, "crlf.txt:/one$/")
	if problems != 0 || !strings.Contains(out, "    1| one\n") {
		t.Fatalf("/one$/ does not serve line 1 without its \\r (problems=%d):\n%q", problems, out)
	}
}

// ADR-065 T1. --grep matches the same lines read serves: `two$` finds line 2
// of a CRLF file.
func TestGrepMatchesADollarAnchoredLineInACRLFFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "crlf.txt", "one\r\ntwo\r\nthree\r\n")
	specs, _, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("two$")})
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 1 {
		t.Fatalf("--grep 'two$' found %d files in a CRLF file, want 1", len(specs))
	}
	var sb strings.Builder
	Run(&sb, root, specs, Options{Numbers: true})
	if !strings.Contains(sb.String(), "    2| two\n") {
		t.Fatalf("--grep 'two$' did not serve line 2:\n%q", sb.String())
	}
}

// ADR-065 T1 guard. A file that MIXES endings is LF on both sides: its stray
// `\r` stays in the line's content, and read still hashes the raw bytes.
func TestAMixedEndingFileKeepsTheStrayCarriageReturnOnBothSides(t *testing.T) {
	root := t.TempDir()
	raw := "a\r\nb\n"
	writeFile(t, root, "mixed.txt", raw)
	out, _ := run(t, root, Options{Numbers: true}, "mixed.txt")
	if !strings.Contains(out, "    1| a\r\n") {
		t.Fatalf("the stray \\r of a mixed file was not kept in line 1:\n%q", out)
	}
	sum := sha256.Sum256([]byte(raw))
	if !strings.Contains(out, "sha "+hex.EncodeToString(sum[:])[:8]) {
		t.Fatalf("the header sha is not the raw bytes' sha:\n%q", out)
	}
}

// ADR-065 T1. ast-grep numbers rows by `\n`, which on a CR-only file is not
// mrw's numbering: its row would be served as a different line. The hit is
// reported as a problem naming the file instead.
func TestAstGrepReportsAHitInACROnlyFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "cr.go", "one\rtwo\rthree\r")
	installFakeAstGrepJSON(t, `[{"file":"cr.go","range":{"start":{"line":1},"end":{"line":1}}}]`, 0)
	specs, problems, err := AstGrep(root, nil, "x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 0 {
		t.Fatalf("a hit in a CR-only file was served: %+v", specs)
	}
	if len(problems) != 1 || problems[0].Path != "cr.go" {
		t.Fatalf("the CR-only hit was not reported as a problem naming cr.go: %+v", problems)
	}
}
