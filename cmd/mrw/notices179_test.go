package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// notices179 is ADR-179 Decision (e)'s list: every notice a read prints, as a
// regexp of its line and an invocation that triggers it among several files or
// specs. The sites `mrw read --grep '"mrw: |"-- [a-z]' --count cmd internal
// --exclude '*_test.go'` finds are accounted for here or in the comment under
// the table: a per-range report (`!!`, WITHHELD, NOT FOUND, the --count total) is
// not a notice, and the rest are not printed by a read.
var notices179 = []struct {
	name  string
	line  string
	env   string
	stdin string
	argv  []string
}{
	{"the long-line hint (T1)", `^mrw: \d+ line\(s\) over 400 columns served whole`, "", "", []string{"read", "l1.txt", "l2.txt", "l3.txt"}},
	{"a bad MRW_MAX_COLS (T1)", `^mrw: MRW_MAX_COLS=abc is not a non-negative integer`, "abc", "", []string{"read", "a.txt", "b.txt"}},
	{"files --outline cannot outline (T2)", `^mrw: --outline reads .go and .md files; no outline for`, "", "", []string{"read", "--outline", "a.py", "b.py", "c.py", "d.go"}},
	{"files --no-comments served whole (T8)", `^mrw: --no-comments reads .go files; served whole:`, "", "", []string{"read", "--no-comments", "a.py", "b.py", "d.go"}},
	{"a quiet read's count", `^-- licensed \d+ line\(s\) in \d+ file\(s\)$`, "", "", []string{"read", "--quiet", "a.txt", "b.txt", "c.txt"}},
	{"the match budget (T7)", `^-- max-matches 1 reached: `, "", "", []string{"read", "--grep", "hit", "--max-matches", "1", "m1.txt", "m2.txt", "m3.txt"}},
	{"what a walk skipped", `^-- skipped: `, "", "", []string{"read", "--grep", "NEEDLE", "."}},
	{"two guards no spec served", `^-- guard not used: a\.txt, b\.txt \(this read did not serve them\)$`, "", "", []string{"read", "--grep", "nomatch-anywhere", "--if-changed", "a.txt=00000000", "--if-served", "b.txt=00000000", "."}},
	{"files served as bytes", `^-- note: u1\.txt is UTF-16 \(BOM FF FE\), u2\.txt is UTF-16 \(BOM FF FE\): served as bytes; a write to them is refused$`, "", "", []string{"read", "u1.txt", "u2.txt", "a.txt"}},
	{"one file's ranges served as bytes", `^-- note: u1\.txt is UTF-16 \(BOM FF FE\): served as bytes; a write to it is refused$`, "", "", []string{"read", "u1.txt:1", "u1.txt:2"}},
	{"one file twice served as bytes", `^-- note: u1\.txt is UTF-16 \(BOM FF FE\): served as bytes; a write to it is refused$`, "", "", []string{"read", "--stat", "u1.txt", "u1.txt"}},
	{"standard input ending in CRLF", `^mrw: standard input ended in CRLF after lines that end in LF`, "", "x\ny\r\n", []string{"read", "-"}},
	{"a plan maker's site count", `^-- planned: 3 site\(s\) in 3 file\(s\)$`, "", "", []string{"read", "--grep", "alpha|bravo|charlie", "--sub", "X", "a.txt", "b.txt", "c.txt"}},
}

// Not printed by a read, so not on the list: the write's CRLF, marks and --create
// notes, skill --dir, startup's move of the legacy directory, run's log rename,
// the -C-before-the-subcommand note, refused176's kept plan, the ledger notices
// (internal/seen, which this record does not own), and mrw_read's own `-- ck`,
// `-- ack`, `-- not licensed` and `-- no checkpoint` lines. The null-device and
// still-licensed notices need the process's standard output to be the null
// device: contract §309 drives them. `mrw: nothing was recorded` (ADR-166) needs a
// state directory mrw cannot write, which ADR-166's own tests make; it is printed
// once, deferred, by construction. The neighbour note and the page notes belong
// to mrw_read's pages (NextLine is off there).

// ADR-179 Decision (e). Each listed notice is triggered by its invocation and
// appears exactly once in it, however many files or specs it concerns, and no
// notice line of any invocation is printed twice.
func TestNoNoticeIsPrintedTwiceInOneInvocation(t *testing.T) {
	long := strings.Repeat("x", 500)
	u16 := "\xff\xfeh\x00i\x00\n\x00"
	root := grepTree(t, map[string]string{
		"l1.txt": long + "\n", "l2.txt": long + "\n", "l3.txt": long + "\n",
		"a.txt": "alpha\n", "b.txt": "bravo\n", "c.txt": "charlie\n",
		"a.py": "x = 1\n", "b.py": "y = 1\n", "c.py": "z = 1\n",
		"d.go":   "package p\n\n// a\n// b\nfunc A() {}\n",
		"m1.txt": "hit 1\nfill\nfill\n", "m2.txt": "hit 2\nfill\nfill\n", "m3.txt": "hit 3\nfill\nfill\n",
		"u1.txt": u16, "u2.txt": u16,
	})
	if err := os.WriteFile(filepath.Join(root, "bin.dat"), []byte("NEEDLE\x00\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "n.txt"), []byte("NEEDLE\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	noticeLine := regexp.MustCompile(`^(mrw: |-- (note|skipped|licensed|guard not used|max-matches|planned))`)
	for _, n := range notices179 {
		t.Setenv("MRW_MAX_COLS", n.env)
		out, errOut, _ := project178Run(t, root, n.stdin, n.argv...)
		all := out + "\n" + errOut
		re := regexp.MustCompile(`(?m)` + n.line)
		if got := len(re.FindAllString(all, -1)); got != 1 {
			t.Errorf("%s: the notice /%s/ appears %d times in mrw %v, want exactly once:\n%s", n.name, n.line, got, n.argv, all)
		}
		seen := map[string]int{}
		for _, l := range strings.Split(all, "\n") {
			if noticeLine.MatchString(l) {
				if seen[l]++; seen[l] == 2 {
					t.Errorf("%s: the line %q is printed twice in mrw %v", n.name, l, n.argv)
				}
			}
		}
	}
	// A guard carried by both --if-changed and --if-served is refused whatever the
	// finder matches (ADR-179 Decision d), under any spelling of the file.
	for _, argv := range [][]string{
		{"read", "--if-changed", "a.txt=00000000", "--if-served", "a.txt=00000000", "a.txt"},
		{"read", "--grep", "nomatch-anywhere", "--if-changed", "./a.txt=00000000", "--if-served", "a.txt=00000000", "."},
	} {
		if out, _, code := project178Run(t, root, "", argv...); code != 2 || !strings.Contains(out, "--if-served and --if-changed") {
			t.Errorf("mrw %v exited %d, want 2 naming both flags:\n%s", argv, code, out)
		}
	}
}
