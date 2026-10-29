package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// terse179Tree holds a.txt, b.txt, c.txt, d.txt and a Go file with a closing
// brace a replace can leave behind; all are read so a plan may edit them.
func terse179Tree(t *testing.T) string {
	t.Helper()
	root := grepTree(t, map[string]string{
		"a.txt": "a1\na2\na3\n",
		"b.txt": "b1\nb2\n",
		"c.txt": "c1\n",
		"d.txt": "d1\n",
		"x.go":  "package a\nfunc A() {\n\tx()\n}\n\nfunc B() {}\n",
	})
	if _, err := readIn(t, root, "a.txt", "b.txt", "c.txt", "d.txt", "x.go"); err != nil {
		t.Fatal(err)
	}
	return root
}

const terse179Plan = "@@ a.txt 1 replace\nA1\n@@ a.txt 3 replace\nA3\n@@ b.txt 2 insert-after\nB3\n"

// ADR-179 Decision (f). Three hunks in two files: a line per file and the
// summary, no per-hunk row and no preview; an unlink, a rename and a dry run
// have their own one-line forms; the default receipt is as it was.
func TestTerseWritePrintsOneLinePerFile(t *testing.T) {
	root := terse179Tree(t)
	plan := planFile(t, terse179Plan)
	out, code := runIn(t, root, "write", "--no-check", "--dry-run", "--terse", plan)
	if want := "dry  a.txt  2 hunk(s)\ndry  b.txt  1 hunk(s)\n3 hunk(s), 2 file(s), 0 failed, 0 advisories — dry run, nothing written\n"; code != 0 || !strings.HasSuffix(out, want) || strings.Contains(out, "ok   ") {
		t.Fatalf("--dry-run --terse exited %d, want exactly %q:\n%q", code, want, out)
	}
	out, code = runIn(t, root, "write", "--no-check", "--terse", plan)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if code != 0 || len(lines) != 3 ||
		!regexp.MustCompile(`^ok   a\.txt  2 hunk\(s\)  -2 \+2  sha [0-9a-f]{8}$`).MatchString(lines[0]) ||
		!regexp.MustCompile(`^ok   b\.txt  1 hunk\(s\)  -0 \+1  sha [0-9a-f]{8}$`).MatchString(lines[1]) ||
		lines[2] != "3 hunk(s), 2 file(s), 0 failed, 0 advisories — applied" {
		t.Errorf("--terse exited %d, want two file lines and the summary:\n%s", code, out)
	}
	if out, code = runIn(t, root, "write", "--no-check", planFile(t, "@@ c.txt - unlink\n@@ d.txt - rename\ne.txt\n")); code != 0 || !strings.Contains(out, "ok   c.txt - unlink") {
		t.Fatalf("the default receipt exited %d, want its per-hunk rows:\n%s", code, out)
	}
	root = terse179Tree(t)
	out, code = runIn(t, root, "write", "--no-check", "--terse", planFile(t, "@@ c.txt - unlink\n@@ d.txt - rename\ne.txt\n"))
	lines = strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if code != 0 || len(lines) != 3 ||
		!regexp.MustCompile(`^ok   c\.txt  unlink  was sha [0-9a-f]{8}$`).MatchString(lines[0]) && !regexp.MustCompile(`^ok   c\.txt  unlink  was sha [0-9a-f]{8}$`).MatchString(lines[1]) ||
		!strings.Contains(out, "ok   d.txt -> e.txt  rename\n") {
		t.Errorf("an unlink and a rename under --terse exited %d:\n%s", code, out)
	}
	// A rename with an edit keeps the edit's tally and the new sha on its line.
	root = terse179Tree(t)
	out, code = runIn(t, root, "write", "--no-check", "--terse", planFile(t, "@@ d.txt 1 replace\nD1\n@@ d.txt - rename\ne.txt\n"))
	if want := regexp.MustCompile(`(?m)^ok   d\.txt -> e\.txt  rename  -1 \+1  sha [0-9a-f]{8}$`); code != 0 || !want.MatchString(out) {
		t.Errorf("an edit and a rename under --terse exited %d, want the tally and sha on the rename line:\n%s", code, out)
	}
	// A write through a symlink names where it went (ADR-105).
	if runtime.GOOS != "windows" {
		root = terse179Tree(t)
		if err := os.Symlink("a.txt", filepath.Join(root, "link.txt")); err != nil {
			t.Fatal(err)
		}
		if _, err := readIn(t, root, "link.txt"); err != nil {
			t.Fatal(err)
		}
		out, code = runIn(t, root, "write", "--no-check", "--terse", planFile(t, "@@ link.txt 1 replace\nL1\n"))
		if code != 0 || !regexp.MustCompile(`(?m)^ok   link\.txt  1 hunk\(s\)  -1 \+1  sha [0-9a-f]{8}  target a\.txt$`).MatchString(out) {
			t.Errorf("a write through a symlink under --terse exited %d, want its target named:\n%s", code, out)
		}
	}
}

// ADR-179 Decision (f). A failed plan prints its FAIL row and the skip rows
// beside it, exactly as the default receipt does; a hunk that moved the
// delimiter count or left a closer below its body prints its balance and
// closer rows, which --quiet drops and --terse keeps.
func TestTerseKeepsEveryFailedHunkAndAdvisory(t *testing.T) {
	root := terse179Tree(t)
	bad := planFile(t, "@@ a.txt 99 replace\nz\n@@ b.txt 1 replace\nz\n")
	plain, pcode := runIn(t, root, "write", "--no-check", bad)
	out, code := runIn(t, root, "write", "--no-check", "--terse", bad)
	if pcode != 1 || code != 1 || out != plain || !strings.Contains(out, "FAIL a.txt 99 replace") || !strings.Contains(out, "skip b.txt 1 replace") {
		t.Errorf("a failed plan under --terse exited %d (default %d), want the default receipt:\n%s\n--- default:\n%s", code, pcode, out, plain)
	}
	adv := planFile(t, "@@ x.go 2-3 replace anchor=\"func A\"\nfunc A() {\n\ty()\n}\n")
	out, code = runIn(t, root, "write", "--no-check", "--terse", adv)
	if code != 0 || !strings.Contains(out, "     balance ") || !strings.Contains(out, "     closer ") || !strings.Contains(out, "ok   x.go  1 hunk(s)") {
		t.Errorf("--terse exited %d, want the balance and closer rows and the file line:\n%s", code, out)
	}
	// What a delete removed (ADR-008) and what a from= copy copied (ADR-167)
	// are the receipt's only account of them, so their hunk rows stay.
	out, code = runIn(t, root, "write", "--no-check", "--terse", planFile(t, "@@ b.txt 1 delete\n@@ c.txt 1 insert-after from=a.txt:2-3\n"))
	if code != 0 || !strings.Contains(out, `ok   b.txt 1 delete  -1 +0 from "b1" to "b1"`) || !strings.Contains(out, `copied a.txt:2-3 from "a2" to "a3"`) {
		t.Errorf("--terse exited %d, want the delete's and the copy's rows:\n%s", code, out)
	}
}

// ADR-179 Decision (f). The check's report and every step's verdict print, and
// the exit code is the check's: 3 when it fails, and 3 for a step that fails.
func TestTerseKeepsTheCheckReportAndItsExit(t *testing.T) {
	t.Setenv("MRW_SHELL", "on")
	needShell(t)
	root := stepsTree(t, "exit 3")
	out, code := writeIn(t, root, "--terse", primed(t, root))
	if code != exitCheckFailed || !strings.Contains(out, "ok   a.go  1 hunk(s)") || !strings.Contains(out, "check FAIL") {
		t.Errorf("a failing check under --terse exited %d, want 3 with its report:\n%s", code, out)
	}
	root = stepsTree(t, "exit 0")
	out, code = writeIn(t, root, "--terse", "--then-sh", "echo stepped; exit 1", primed(t, root))
	if code != exitCheckFailed || !strings.Contains(out, "ok   a.go  1 hunk(s)") || !strings.Contains(out, "stepped") {
		t.Errorf("a failing step under --terse exited %d, want 3 with its verdict:\n%s", code, out)
	}
}

// ADR-179 Decision (f). --json is the same document with and without --terse,
// and --terse beside --quiet is a usage error.
func TestTerseLeavesTheJSONReceiptUnchanged(t *testing.T) {
	plain, code := runIn(t, terse179Tree(t), "write", "--no-check", "--json", planFile(t, terse179Plan))
	if code != 0 {
		t.Fatalf("the default --json write exited %d:\n%s", code, plain)
	}
	terse, code := runIn(t, terse179Tree(t), "write", "--no-check", "--json", "--terse", planFile(t, terse179Plan))
	var a, b map[string]any
	if code != 0 || json.Unmarshal([]byte(plain), &a) != nil || json.Unmarshal([]byte(terse), &b) != nil {
		t.Fatalf("--json --terse exited %d:\n%s", code, terse)
	}
	if h, ok := b["hunks"].([]any); !ok || len(h) != 3 {
		t.Errorf("--terse changed the hunks of the JSON receipt: %v", b["hunks"])
	}
	for _, k := range []string{"pattern", "files", "root"} {
		delete(a, k)
		delete(b, k)
	}
	if !reflect.DeepEqual(a, b) {
		t.Errorf("--terse changed the JSON receipt:\n%v\n%v", a, b)
	}
	if out, code := runIn(t, terse179Tree(t), "write", "--no-check", "--terse", "--quiet", planFile(t, terse179Plan)); code != 2 || !strings.Contains(out, "--terse") || !strings.Contains(out, "--quiet") {
		t.Errorf("--terse --quiet exited %d, want 2 naming both:\n%s", code, out)
	}
}
