package main

import (
	"runtime"
	"strings"
	"testing"
)

// An address pattern that holds a space has to be quoted, or the shell hands
// mrw two arguments: `mrw read f.go:/func Foo/` arrives as `f.go:/func` and
// `Foo/`, and the refusal said only that "/func" is never closed. Counted
// 2026-11-19 across Claude Code transcripts outside this repository (a one-off
// tally, aggregates only): of 100 "never closed" refusals of a read, 85 named
// a spec that the arguments after it, joined by spaces, complete. The refusal
// now says the spec was split and prints it whole, quoted; on a Windows build,
// which may be answering cmd.exe, it says to quote it and prints no spelling.
func TestASpecTheShellSplitAtASpaceIsNamed(t *testing.T) {
	run, _ := ifChangedRun(t, map[string]string{"f.go": "package a\n\nfunc Foo Bar() {\n}\n", "g.go": "package a\n", "list.txt": "f.go:/func\\nFoo/\\n", "one.txt": "f.go:/func\\n"})
	const said = "; the shell split this spec at a space: quote it, "
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"f.go:/func", "Foo/"}, said + "'f.go:/func Foo/'"},
		{[]string{"f.go:/func", "Foo", "Bar/"}, said + "'f.go:/func Foo Bar/'"},
		{[]string{"f.go:/func", "Foo/,+1"}, said + "'f.go:/func Foo/,+1'"},
		{[]string{"g.go", "f.go:/func", "Foo/,/^}/"}, said + "'f.go:/func Foo/,/^}/'"},
		{[]string{"f.go:/it's", "Foo/"}, "; the shell split this spec at a space: quote the whole spec"},
		{[]string{"f.go:/café", "Foo/"}, "; the shell split this spec at a space: quote the whole spec"},
		{[]string{"f.go:/a\x1bb", "Foo/"}, "; the shell split this spec at a space: quote the whole spec"},
		{[]string{"--", "-f.go:/func", "Foo/"}, "; the shell split this spec at a space: quote the whole spec"},
		{[]string{"f.go:/a", "b", "c", "d", "e", "f", "g", "h", "i/"}, said + "'f.go:/a b c d e f g h i/'"},
	} {
		if runtime.GOOS == "windows" {
			c.want = "; the shell split this spec at a space: quote the whole spec"
		}
		out, errOut, code := run("", append([]string{"read"}, c.args...)...)
		if code != exitUsage || !strings.HasSuffix(strings.TrimSpace(errOut), c.want) || out != "" {
			t.Errorf("read %q: exit %d, stdout %q, want exit %d ending in %q:\n%s", c.args, code, out, exitUsage, c.want, errOut)
		}
		if !strings.Contains(errOut, `pattern "/`) || !strings.Contains(errOut, "is never closed") {
			t.Errorf("read %q: the refusal lost what it said before:\n%s", c.args, errOut)
		}
	}
	// What is not a split spec keeps the refusal it had: a pattern that is not
	// closed by what follows, one with nothing after it, one more than eight
	// arguments away from its end, and a join that parses as another file or
	// another kind of address.
	for _, args := range [][]string{{"f.go:/func Foo"}, {"f.go:/func", "g.go"}, {"f.go:/func", "Foo/x"}, {"f.go:/func"},
		{"f.go:/a", "b", "c", "d", "e", "f", "g", "h", "i", "j/"},
		{"f.go:nope", "g.go:1"}, {"f.go:/func", "g.go:1"}, {"f.go:/func", "Foo/,3"}} {
		out, errOut, code := run("", append([]string{"read"}, args...)...)
		if code != exitUsage || out != "" || strings.Contains(errOut, "the shell split") {
			t.Errorf("read %q: exit %d, want exit %d with no word of a split:\n%s", args, code, exitUsage, errOut)
		}
	}
	// Two lines of a --files-from list were not split by a shell, and neither
	// was a line of it and an argument typed beside it.
	for _, args := range [][]string{{"--files-from", "list.txt"}, {"--files-from", "one.txt", "Foo/"}} {
		_, errOut, code := run("", append([]string{"read"}, args...)...)
		if code == exitUsage && !strings.Contains(errOut, "the shell split") {
			continue
		}
		t.Errorf("read %q: exit %d, want exit %d with no word of a split:\n%s", args, code, exitUsage, errOut)
	}
	// Nor was a working-set entry, whatever is typed after its number.
	if _, errOut, code := run("", "iter", "add", "f.go:/func"); code != 0 {
		t.Fatalf("iter add: exit %d\n%s", code, errOut)
	}
	if _, errOut, code := run("", "read", "@1", "Foo/"); code != exitUsage || strings.Contains(errOut, "the shell split") {
		t.Errorf("read @1 Foo/: exit %d, want exit %d with no word of a split:\n%s", code, exitUsage, errOut)
	}
	// The pair: the spec the refusal prints is served, and only its line.
	if out, errOut, code := run("", "read", "f.go:/func Foo/"); code != 0 || !strings.Contains(out, "func Foo Bar() {") || strings.Contains(out, "package a") {
		t.Errorf("read 'f.go:/func Foo/': exit %d, want the line served:\n%s%s", code, out, errOut)
	}
}
