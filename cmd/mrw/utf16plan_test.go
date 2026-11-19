package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

// Windows PowerShell 5.1 writes UTF-16 when a command's output is redirected
// with `>`, so `mrw read --sub … > plan.txt` leaves a plan mrw cannot read.
// The refusal was one "text before the first @@ header" error per line, each
// quoting the line as hex, and never said UTF-16 or named the redirection
// (found by a Windows peer session, 2026-11-19). It is now one sentence.
func TestAPlanFileInUTF16IsRefusedInOneSentence(t *testing.T) {
	run, root := ifChangedRun(t, map[string]string{"f.txt": "one\ntwo\n"})
	if out, errOut, code := run("", "read", "f.txt"); code != 0 {
		t.Fatalf("read: exit %d\n%s%s", code, out, errOut)
	}
	const plan = "@@ f.txt 1 replace\nONE\n@@ f.txt 2 replace\nTWO\n"
	units := utf16.Encode([]rune(plan))
	le, be := []byte{0xff, 0xfe}, []byte{0xfe, 0xff}
	var bare []byte
	for _, u := range units {
		le = append(le, byte(u), byte(u>>8))
		be = append(be, byte(u>>8), byte(u))
		bare = append(bare, byte(u), byte(u>>8))
	}
	for name, body := range map[string][]byte{"little endian with a mark": le, "big endian with a mark": be, "little endian with no mark": bare} {
		p := filepath.Join(t.TempDir(), "plan.txt")
		if err := os.WriteFile(p, body, 0o644); err != nil {
			t.Fatal(err)
		}
		for _, how := range []string{"file", "stdin"} {
			args, stdin := []string{"write", p}, ""
			if how == "stdin" {
				args, stdin = []string{"write", "-"}, string(body)
			}
			out, errOut, code := run(stdin, args...)
			lines := strings.Split(strings.TrimSpace(errOut), "\n")
			last := lines[len(lines)-1]
			if code != exitUsage || out != "" || !strings.Contains(last, "UTF-16") || !strings.Contains(last, "Out-File -Encoding utf8") {
				t.Errorf("%s, %s: exit %d, stdout %q, want exit %d and one sentence naming UTF-16 and a way out:\n%s", name, how, code, out, exitUsage, errOut)
			}
			if strings.Contains(errOut, "text before the first @@ header") || strings.Contains(errOut, `\x00`) {
				t.Errorf("%s, %s: the refusal still dumps the lines:\n%s", name, how, errOut)
			}
		}
	}
	// The two foreign formats read the whole document first; they say the same.
	p16 := filepath.Join(t.TempDir(), "doc.txt")
	if err := os.WriteFile(p16, le, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"apply_patch", "search_replace"} {
		out, errOut, code := run("", "write", "--format", format, p16)
		if code != exitUsage || out != "" || !strings.Contains(errOut, "UTF-16") || strings.Contains(errOut, `\x00`) {
			t.Errorf("--format %s: exit %d, stdout %q, want exit %d and the sentence:\n%s", format, code, out, exitUsage, errOut)
		}
	}
	if got, _ := os.ReadFile(filepath.Join(root, "f.txt")); string(got) != "one\ntwo\n" {
		t.Errorf("a refused plan changed the file: %q", got)
	}
	// A piped plan that is refused is kept whole (ADR-176), not cut where the
	// check stopped reading; and the refusal is counted as a plan that did not parse.
	long := append([]byte{}, le...)
	for len(long) < 200000 {
		long = append(long, le[2:]...)
	}
	_, errOut, code := run(string(long), "write", "--json", "-")
	kept := ""
	for _, f := range strings.Fields(errOut) {
		if strings.HasSuffix(f, ".plan") {
			kept = f
		}
	}
	if b, err := os.ReadFile(kept); code != exitUsage || err != nil || len(b) != len(long) {
		t.Errorf("a refused piped plan of %d bytes: exit %d, kept %q holds %d (%v):\n%s", len(long), code, kept, len(b), err, errOut)
	}
	if out, _, _ := run("", "stats", "--json"); !strings.Contains(out, `"refused_parse"`) || strings.Contains(out, `"refused_parse": 0`) || strings.Contains(out, `"refused_parse":0`) {
		t.Errorf("the refusals are not counted as plans that did not parse:\n%s", out)
	}
	// A NUL is valid UTF-8, and the parsers allow one in a comment: such a plan is
	// read, not called UTF-16.
	if out, errOut, code := run("#\x00x\x00\n@@ f.txt /^two$/ replace\nTWO\n", "write", "-"); code != 0 || strings.Contains(errOut, "UTF-16") {
		t.Errorf("a comment holding NULs before a hunk: exit %d\n%s%s", code, out, errOut)
	}
	// What is not UTF-16 is read as before: the same plan in UTF-8, with a byte
	// order mark too, and a line that merely holds a NUL is not called UTF-16.
	p := filepath.Join(t.TempDir(), "plan8.txt")
	if err := os.WriteFile(p, []byte("\xef\xbb\xbf@@ f.txt /^one$/ replace\nONE\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, errOut, code := run("", "write", p); code != 0 {
		t.Errorf("a UTF-8 plan with a mark: exit %d\n%s%s", code, out, errOut)
	}
	// A plan shorter than four bytes is judged by what it holds, not past its end.
	for _, short := range []string{"@", "@@", "@@\n", "x\x00y"} {
		if _, errOut, code := run(short, "write", "-"); code == 0 || strings.Contains(errOut, "UTF-16") || strings.Contains(errOut, "panic") {
			t.Errorf("a %d-byte plan: exit %d, want a refusal that is not about UTF-16:\n%s", len(short), code, errOut)
		}
	}
	if _, errOut, code := run("@@ f.txt 2 replace\nT\x00O\n", "write", "-"); strings.Contains(errOut, "UTF-16") {
		t.Errorf("a NUL inside a body line was called UTF-16 (exit %d):\n%s", code, errOut)
	}
}
