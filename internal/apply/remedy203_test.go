package apply

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
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
		{false, "1.0.0", ""}, // a word that starts with a digit is described: PowerShell's numeric grammar is wide (1.e2, 5., 0x, 2kb, 1_000)
		{false, "1.e2", ""},
		{false, "5.", ""},
		{false, ".5", ""},
		{false, "2024-reports", ""},
		{false, "_/dev/null", "_/dev/null"}, // a POSIX shell converts nothing
		{false, "v1.10", "v1.10"},
		{false, "1.10/x", "1.10/x"},
		{true, "r1", "r1"},
		// Git Bash (MSYS), find_path_start_and_type in msys2_path_conv.cc: it skips leading bytes that are not
		// alphanumeric and not one of / \ : - . (in the plain set, only `_`), then rewrites the rest if it begins
		// with `/` (`_/worktree`, `__/x`, `_//x`, UNC); `sub/r1`, `a//b`, `tmp/x`, `sub/dev/null` and `./x` are left alone.
		{true, "sub/r1", "sub/r1"},
		{true, `sub\r1`, "sub/r1"},
		{true, `_\worktree`, ""},
		{true, "_/worktree", ""},
		{true, "__/x", ""},
		{true, "_//x", ""},
		{true, "_worktree", "_worktree"},
		{true, "_a/b", "_a/b"},
		{true, "worktree", "worktree"},
		{true, "tmp/x", "tmp/x"},
		{true, "sub/dev/null", "sub/dev/null"},
		{true, "./x", "./x"},
		{true, "../x", "../x"},
		{true, "a/b:/c", ""}, // a `:` before a `/` after an earlier `/` is a path list: not in the plain set
		{true, "sub/x:.y", ""},
		{true, `C:\Users\x\r1`, "C:/Users/x/r1"},
		{true, `c:\r1`, "c:/r1"},
		{true, "C:/Users/x/r1", "C:/Users/x/r1"},
		{true, `C:\Users\RUNNER~1\AppData\Local\Temp\r1`, "C:/Users/RUNNER~1/AppData/Local/Temp/r1"},
		{true, `\\server\share\r1`, ""}, // MSYS rewrites it
		{true, `\Users\x\r1`, ""},
		{true, "/Users/x/r1", ""},
		{true, "~r1", ""},
		{true, "_/dev/null", ""},
		{true, "a//b", "a//b"},
		{true, "sub/", "sub/"},
		{true, `C:\dev\x`, "C:/dev/x"}, // a drive path is a Windows path already: MSYS leaves it
		{true, "C:/x:y", ""},           // a drive path is copied verbatim only with no later colon
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
	// A path of the plan is judged the same way, and printed with slashes on Windows; a path with a
	// directory is commanded there, as a root with one is (the common case: internal/apply/x.go).
	if got := remedyFor(true, "", `sub\F.go`, 1, 4); got != `mrw read --max-cols 0 sub/F.go:1-4` {
		t.Errorf("a backslash path on Windows: %q", got)
	}
	if got := remedyFor(true, "", `internal/apply/x.go`, 1, 4); got != `mrw read --max-cols 0 internal/apply/x.go:1-4` {
		t.Errorf("a path with directories on Windows: %q", got)
	}
	if got := remedyFor(true, "", `_/x.go`, 1, 4); got != `a read that shows lines 1-4 of '_/x.go'` {
		t.Errorf("a path that MSYS would convert on Windows: %q", got)
	}
	if got := remedyFor(false, "", `sub\F.go`, 1, 4); got != `a read that shows lines 1-4 of 'sub\F.go'` {
		t.Errorf("a backslash path on POSIX: %q", got)
	}
}

// cmd.exe refuses a command line over 8,191 characters: a command longer than the
// bound is described on Windows, where it could not run, and printed elsewhere.
func TestACommandCmdCannotRunIsDescribedOnWindows(t *testing.T) {
	long := "C:/" + strings.Repeat("a", 8000)
	if got := remedyFor(true, long, "F.go", 1, 4); !strings.HasPrefix(got, "a read that shows lines 1-4 of 'F.go' under the root 'C:/aaa") {
		t.Errorf("a root of 8,000 characters on Windows: %.80q", got)
	}
	posix := "sub/" + strings.Repeat("a", 8000)
	if got := remedyFor(false, posix, "F.go", 1, 4); !strings.HasPrefix(got, "mrw --root sub/aaa") {
		t.Errorf("a root of 8,000 characters on POSIX: %.80q", got)
	}
	short := "C:/" + strings.Repeat("a", 7000)
	if got := remedyFor(true, short, "F.go", 1, 4); !strings.HasPrefix(got, "mrw --root C:/aaa") {
		t.Errorf("a root of 7,000 characters on Windows: %.80q", got)
	}
}

// A name in a description is single-quoted as written, and Go-quoted wherever a
// character could split a line or mislead a reader: a control or format character
// (U+2028, U+2029, U+0085, a bidi mark), or bytes that are not UTF-8, which JSON
// would otherwise replace silently.
func TestADescriptionEscapesWhatCouldSplitALineOrMislead(t *testing.T) {
	for name, want := range map[string]string{
		"café r1":        "'café r1'",
		"a\u2028b r":     `"a\u2028b r"`,
		"a\u2029b r":     `"a\u2029b r"`,
		"a\u0085b r":     `"a\u0085b r"`,
		"a\u202eb r":     `"a\u202eb r"`, // right-to-left override
		"a\u200eb r":     `"a\u200eb r"`,
		"a\xffb r":       `"a\xffb r"`,
		"it's r":         `"it's r"`,
		"a\nb r":         `"a\nb r"`,
		`C:\my dir\r`:    `'C:\my dir\r'`,
		"\u00a0r1 and r": "'\u00a0r1 and r'", // a no-break space is printable
	} {
		got := remedyFor(false, name, "F.go", 1, 4)
		if wantFull := "a read that shows lines 1-4 of 'F.go' under the root " + want; got != wantFull {
			t.Errorf("%q: %q, want %q", name, got, wantFull)
		}
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
