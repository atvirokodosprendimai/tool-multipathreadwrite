package apply

import (
	"fmt"
	"runtime"
	"strconv"
	"testing"
)

// named is how a description names a file or a root: single-quoted as it is, so it
// copies as written, and Go-quoted only where a control byte or a quote forces it.
func named(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] == 0x7f || s[i] == '\'' {
			return strconv.Quote(s)
		}
	}
	return "'" + s + "'"
}

// The stale-number refusal prints a read to paste only for a path that needs no
// quoting in any shell mrw meets: single quotes do not quote in cmd.exe, so the
// command is never quoted, and a path that would need quoting gets a
// description instead (ADR-203), naming the file.
func TestTheStaleNumberRemedyIsACommandOnlyForAPlainPath(t *testing.T) {
	desc := func(p string) string { return "a read that shows line 8 of " + named(p) }
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
// platform's. On Windows a drive root needs no quoting in cmd, PowerShell 7 or
// Windows PowerShell 5.1, but Git Bash eats a backslash, so it is printed with
// slashes, which all four run and Go reads as the same directory; a root with a
// leading separator (UNC, or rooted without a drive) is described, since MSYS
// rewrites it. Every space and metacharacter is described too, and a word
// PowerShell would turn into a number. Both remedies (the stale number's one line
// and the line after's range) go through the one function.
func TestARootIsPrintedInTheCommandWheneverNoShellNeedsItQuoted(t *testing.T) {
	// command is the root as the command prints it, "" when it is described.
	cases := []struct {
		windows bool
		root    string
		command string
	}{
		{false, "r1", "r1"},
		{false, "sub/r1", "sub/r1"},
		{false, "/Users/x/r1", "/Users/x/r1"},
		{false, "../r1", "../r1"},
		{false, `r1\sub`, ""}, // a backslash is a character, not a separator, in a POSIX shell
		{false, `C:\Users\x\r1`, ""},
		{false, "C:/Users/x/r1", ""},
		{false, "/Users/RUNNER~1/r1", "/Users/RUNNER~1/r1"}, // a tilde inside a word is literal in every shell
		{false, "~r1", ""},                                  // at the start it expands
		{false, "r1~", "r1~"},
		{false, "1.10", ""}, // PowerShell would read a number: 1.1
		{false, "0x10", ""},
		{false, "007", ""},
		{false, "2kb", ""},
		{false, "1e3", ""},
		{false, "1.0.0", "1.0.0"}, // two dots: not a number
		{false, "v1.10", "v1.10"},
		{false, "1.10/x", "1.10/x"},
		{true, "r1", "r1"},
		{true, "sub/r1", "sub/r1"},
		{true, `sub\r1`, "sub/r1"},
		{true, `C:\Users\x\r1`, "C:/Users/x/r1"},
		{true, `c:\r1`, "c:/r1"},
		{true, "C:/Users/x/r1", "C:/Users/x/r1"},
		{true, `C:\Users\RUNNER~1\AppData\Local\Temp\r1`, "C:/Users/RUNNER~1/AppData/Local/Temp/r1"},
		{true, `\\server\share\r1`, ""}, // MSYS rewrites it
		{true, `\Users\x\r1`, ""},
		{true, "/Users/x/r1", ""},
		{true, "~r1", ""},
		{true, `C:\Users\my dir\r1`, ""},
		{true, `C:r1`, ""}, // drive-relative: not an absolute path, and `:` is not plain
		{true, `C:`, ""},
		{true, `x:y\r1`, ""},
		{true, "-r1", ""},
		{true, "1.10", ""},
		{true, "", ""},
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
			case c.root == "":
				want = "mrw read --max-cols 0 F.go:" + spans
			case c.command != "":
				want = "mrw --root " + c.command + " read --max-cols 0 F.go:" + spans
			default:
				want = "a read that shows " + words + " of 'F.go' under the root " + named(c.root)
			}
			if got != want {
				t.Errorf("%s, windows %v, root %q: %q, want %q", f.name, c.windows, c.root, got, want)
			}
		}
	}
	// Each metacharacter, alone and inside an otherwise plain root, on both platforms.
	for _, windows := range []bool{false, true} {
		for _, ch := range "&|<>^%!'\"();,=$`{}*?[@# " {
			for _, root := range []string{"r" + string(ch) + "1", string(ch) + "r1", "r1" + string(ch)} {
				if got := remedyFor(windows, root, "F.go", 1, 4); got != "a read that shows lines 1-4 of 'F.go' under the root "+named(root) {
					t.Errorf("windows %v, root %q is commanded or misdescribed: %q", windows, root, got)
				}
			}
		}
	}
	// A path of the plan is judged the same way, and printed with slashes on Windows.
	if got := remedyFor(true, "", `sub\F.go`, 1, 4); got != `mrw read --max-cols 0 sub/F.go:1-4` {
		t.Errorf("a backslash path on Windows: %q", got)
	}
	if got := remedyFor(false, "", `sub\F.go`, 1, 4); got != `a read that shows lines 1-4 of 'sub\F.go'` {
		t.Errorf("a backslash path on POSIX: %q", got)
	}
}

// The platform is the build's: a drive root with backslashes is commanded, in its
// slash form, exactly when this build is Windows. Nothing else asks the question of
// the real GOOS, and a build that always said one or the other would pass the table.
func TestTheRemedyAsksThePlatformItRunsOn(t *testing.T) {
	got := staleRemedySpan(`C:\Users\x\r1`, "F.go", 1, 4)
	want := `a read that shows lines 1-4 of 'F.go' under the root 'C:\Users\x\r1'`
	if runtime.GOOS == "windows" {
		want = "mrw --root C:/Users/x/r1 read --max-cols 0 F.go:1-4"
	}
	if got != want {
		t.Errorf("on %s: %q, want %q", runtime.GOOS, got, want)
	}
}
