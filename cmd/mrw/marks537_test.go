package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Issue #537. Windows PowerShell 5.1 with a UTF-8 $OutputEncoding that has a
// preamble pipes two byte order marks before the plan, and the first header
// was refused. Every leading mark is dropped and a count above one is named;
// one mark stays silent, as before. The refused plan is kept (ADR-176) and
// must rerun by its path.
func TestAPlanWithTwoLeadingByteOrderMarksApplies(t *testing.T) {
	const bom = "\xef\xbb\xbf"
	k := newKeptRig(t)
	for _, c := range []struct {
		name  string
		marks int
		note  string
	}{
		{"one", 1, ""},
		{"two", 2, "the plan begins with 2 UTF-8 byte order marks, all dropped"},
		{"three", 3, "the plan begins with 3 UTF-8 byte order marks, all dropped"},
	} {
		k.reset()
		k.file("f.txt", "a\nb\n")
		k.run("", nil, "read", "--quiet", "f.txt")
		_, errOut, code := k.run(strings.Repeat(bom, c.marks)+"@@ f.txt 1 replace body=1\nALPHA\n", nil, "write", "--no-check", "-")
		if code != 0 || k.read("f.txt") != "ALPHA\nb\n" {
			t.Errorf("%s: exit %d, f.txt %q, want 0 and ALPHA\n%s", c.name, code, k.read("f.txt"), errOut)
		}
		if c.note == "" && strings.Contains(errOut, "byte order mark") {
			t.Errorf("%s: a single mark is silent, stderr:\n%s", c.name, errOut)
		}
		if c.note != "" && !strings.Contains(errOut, "mrw: "+c.note) {
			t.Errorf("%s: stderr does not say %q:\n%s", c.name, c.note, errOut)
		}
	}
}

func TestAKeptPlanWithTwoByteOrderMarksRerunsByItsPath(t *testing.T) {
	const bom = "\xef\xbb\xbf"
	k := newKeptRig(t)
	k.file("f.txt", "a\nb\n")
	plan := bom + bom + "@@ f.txt 1 replace body=1\nALPHA\n"
	// f.txt was never read, so the plan is refused and kept.
	if _, _, code := k.run(plan, nil, "write", "--no-check", "-"); code == 0 {
		t.Fatal("an unread file was written")
	}
	path, got := k.onlyKept("two marks")
	if got != plan {
		t.Fatalf("kept bytes %q, want the plan %q", got, plan)
	}
	k.run("", nil, "read", "--quiet", "f.txt")
	if _, e, code := k.run("", nil, "write", "--no-check", path); code != 0 || k.read("f.txt") != "ALPHA\nb\n" {
		t.Errorf("the kept plan did not rerun: exit %d, f.txt %q\n%s", code, k.read("f.txt"), e)
	}
}

// Review of #539. The two foreign formats took no leading mark at all; both now
// take one silently and several with the count named.
func TestTheForeignFormatsTakeLeadingByteOrderMarksFromStandardInput(t *testing.T) {
	const bom = "\xef\xbb\xbf"
	const patch = "*** Begin Patch\n*** Add File: n.txt\n+hi\n*** End Patch\n"
	const sr = "f.txt\n<<<<<<< SEARCH\na\n=======\nA\n>>>>>>> REPLACE\n"
	k := newKeptRig(t)
	for _, c := range []struct{ format, doc, file, want string }{
		{"apply_patch", patch, "n.txt", "hi\n"},
		{"search_replace", sr, "f.txt", "A\nb\n"},
	} {
		for marks := 1; marks <= 2; marks++ {
			k.reset()
			k.file("f.txt", "a\nb\n")
			_ = os.Remove(filepath.Join(k.root, "n.txt"))
			k.run("", nil, "read", "--quiet", "f.txt")
			_, errOut, code := k.run(strings.Repeat(bom, marks)+c.doc, nil, "write", "--no-check", "--format", c.format, "-")
			b, _ := os.ReadFile(filepath.Join(k.root, c.file)) // n.txt is absent while the format is refused
			if code != 0 || string(b) != c.want {
				t.Errorf("%s, %d mark(s): exit %d, %s %q\n%s", c.format, marks, code, c.file, b, errOut)
			}
			named := strings.Contains(errOut, "mrw: the "+c.format+" document begins with 2 UTF-8 byte order marks, all dropped")
			if (marks == 2) != named || (marks == 1 && strings.Contains(errOut, "byte order mark")) {
				t.Errorf("%s, %d mark(s): the count note is wrong:\n%s", c.format, marks, errOut)
			}
		}
	}
}
