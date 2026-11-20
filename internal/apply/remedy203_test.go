package apply

import (
	"fmt"
	"strconv"
	"testing"
)

// The stale-number refusal prints a read to paste only for a path that needs no
// quoting in any shell mrw meets: single quotes do not quote in cmd.exe, so the
// command is never quoted, and a path that would need quoting gets a
// description instead (ADR-203), naming the file.
func TestTheStaleNumberRemedyIsACommandOnlyForAPlainPath(t *testing.T) {
	desc := func(p string) string { return "a read that shows line 8 of " + strconv.Quote(p) }
	for p, want := range map[string]string{
		"f.txt":            "mrw read --max-cols 0 f.txt:8",
		"sub/dir/a_b-c.go": "mrw read --max-cols 0 sub/dir/a_b-c.go:8",
		"a":                desc("a"),
		"ab":               "mrw read --max-cols 0 ab:8",
		"my file.txt":      desc("my file.txt"),
		"it's.txt":         desc("it's.txt"),
		"-f.txt":           desc("-f.txt"),
		"a$b.txt":          desc("a$b.txt"),
		"a%b.txt":          desc("a%b.txt"),
		"café.txt":         desc("café.txt"),
		"a\x1bb.txt":       desc("a\x1bb.txt"),
		"":                 desc(""),
	} {
		if got := remedyFor(false, "", p, 8, 8); got != want {
			t.Errorf("staleRemedy(%q, 8) = %q, want %q", p, got, want)
		}
	}
}

// Issue #753. Whether a root is printed in the command or described is the
// platform's: on Windows an absolute root with a drive letter and backslashes needs
// no quoting in cmd, PowerShell 7 or Windows PowerShell 5.1, so it is commanded;
// every space and metacharacter still gets the description, which names the file
// and the root. Both remedies (the stale number's one line and the line after's
// range) go through the one function.
func TestARootIsPrintedInTheCommandWheneverNoShellNeedsItQuoted(t *testing.T) {
	const bad = "&|<>^%!'\"();,=$`{}~*?[@# "
	cases := []struct {
		windows bool
		root    string
		command bool
	}{
		{false, "r1", true},
		{false, "sub/r1", true},
		{false, "/Users/x/r1", true},
		{false, "../r1", true},
		{false, `r1\sub`, false}, // a backslash is a character, not a separator, in a POSIX shell
		{false, `C:\Users\x\r1`, false},
		{false, "C:/Users/x/r1", false},
		{true, "r1", true},
		{true, "sub/r1", true},
		{true, `sub\r1`, true},
		{true, `C:\Users\x\r1`, true},
		{true, `c:\r1`, true},
		{true, "C:/Users/x/r1", true},
		{true, `\\server\share\r1`, true},
		{true, `C:\Users\my dir\r1`, false},
		{true, `C:r1`, false}, // drive-relative: not an absolute path, and `:` is not plain
		{true, `C:`, false},
		{true, `x:y\r1`, false},
		{true, "-r1", false},
		{true, "", true}, // no root named: the command names none
	}
	for _, c := range cases {
		for _, f := range []struct {
			name     string
			from, to int
		}{{"stale number", 2, 2}, {"line after", 1, 4}} {
			got := remedyFor(c.windows, c.root, "F.go", f.from, f.to)
			spans := strconv.Itoa(f.from)
			words := "line " + spans
			if f.from != f.to {
				spans, words = fmt.Sprintf("%d-%d", f.from, f.to), fmt.Sprintf("lines %d-%d", f.from, f.to)
			}
			var want string
			switch {
			case c.command && c.root == "":
				want = "mrw read --max-cols 0 F.go:" + spans
			case c.command:
				want = "mrw --root " + c.root + " read --max-cols 0 F.go:" + spans
			case c.root == "":
				want = "a read that shows " + words + " of \"F.go\""
			default:
				want = "a read that shows " + words + " of \"F.go\" under the root " + strconv.Quote(c.root)
			}
			if got != want {
				t.Errorf("%s, windows %v, root %q: %q, want %q", f.name, c.windows, c.root, got, want)
			}
		}
	}
	// Each metacharacter, alone and inside an otherwise plain root, on both platforms.
	for _, windows := range []bool{false, true} {
		for _, ch := range bad {
			for _, root := range []string{"r" + string(ch) + "1", string(ch) + "r1", "r1" + string(ch)} {
				if got := remedyFor(windows, root, "F.go", 1, 4); got != "a read that shows lines 1-4 of \"F.go\" under the root "+strconv.Quote(root) {
					t.Errorf("windows %v, root %q is commanded or misdescribed: %q", windows, root, got)
				}
			}
		}
	}
	// A path of the plan is judged the same way.
	if got := remedyFor(true, "", `sub\F.go`, 1, 4); got != `mrw read --max-cols 0 sub\F.go:1-4` {
		t.Errorf("a backslash path on Windows: %q", got)
	}
}
