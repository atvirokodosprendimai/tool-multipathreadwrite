package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// rwStream is ast-grep --json=stream for the given matches of file in body:
// one object per line, the fields ast-grep 0.45.3 reports.
func rwStream(t *testing.T, file, body string, matches ...[2]string) string {
	t.Helper()
	var sb strings.Builder
	for _, m := range matches {
		start := strings.Index(body, m[0])
		if start < 0 {
			t.Fatalf("%q is not in %q", m[0], body)
		}
		obj := map[string]any{
			"text":        m[0],
			"range":       map[string]any{"byteOffset": map[string]int{"start": start, "end": start + len(m[0])}},
			"file":        file,
			"replacement": m[1],
		}
		b, err := json.Marshal(obj)
		if err != nil {
			t.Fatal(err)
		}
		sb.Write(b)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// ADR-161 Decision 3. `read --ast-grep P --rewrite R` prints ast-grep's own
// rewrite as a plan and licenses it, and `write --ast-grep P --rewrite R`
// applies the same plan in one command. Overlapping matches, a usage mistake
// and no match all change nothing.
func TestARewriteReadAndWriteApplyAstGrepsResult(t *testing.T) {
	const a = "package a\n\nfunc f() {\n\tfoo(1, 2)\n\tx := foo(\n\t\t3,\n\t\t4)\n\t_ = foo(foo(5, 6), 7)\n\ts := \"naïve\"; foo(s, \"ŝ\")\n\t_ = x\n}\n"
	const want = "package a\n\nfunc f() {\n\tbar(2, 1)\n\tx := bar(4, 3)\n\t_ = foo(foo(5, 6), 7)\n\ts := \"naïve\"; bar(\"ŝ\", s)\n\t_ = x\n}\n"
	root, run := subRun(t)
	subTree(t, root, map[string]string{"a.go": a, "other.go": "package a\n"})
	args := installRecordingAstGrep(t, rwStream(t, "a.go", a,
		[2]string{"foo(1, 2)", "bar(2, 1)"},
		[2]string{"foo(\n\t\t3,\n\t\t4)", "bar(4, 3)"},
		[2]string{`foo(s, "ŝ")`, `bar("ŝ", s)`}))
	unchanged := func(why string) {
		t.Helper()
		if got := subFile(t, root, "a.go"); got != a {
			t.Errorf("%s changed a.go to %q", why, got)
		}
	}

	out, errOut, code := run("", "read", "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", ".")
	if code != 0 || !strings.Contains(out, "@@ a.go 4-7 replace sha=") || !strings.Contains(out, "anchor=\"\tfoo(1, 2)\"") || strings.Contains(out, "-- planned") {
		t.Errorf("read: exit %d, want 0 and the plan on stdout alone:\n%s%s", code, out, errOut)
	}
	if !strings.Contains(errOut, "-- planned: 2 site(s) in 1 file(s)") {
		t.Errorf("read: stderr lacks the count:\n%s", errOut)
	}
	got, _ := os.ReadFile(args)
	for _, token := range []string{"--rewrite=bar($B, $A)", "--json=stream", "foo($A, $B)"} {
		if !strings.Contains(string(got), token) {
			t.Errorf("ast-grep was started with %q, want %q among its arguments", got, token)
		}
	}
	unchanged("a read")

	// the plan is licensed by the read that printed it
	if wout, werr, wcode := run(out, "write", "--no-check", "-"); wcode != 0 || subFile(t, root, "a.go") != want {
		t.Errorf("read | write: exit %d, a.go %q, want %q:\n%s%s", wcode, subFile(t, root, "a.go"), want, wout, werr)
	}
	subTree(t, root, map[string]string{"a.go": a})
	if wout, werr, wcode := run("", "write", "--no-check", "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", "."); wcode != 0 || subFile(t, root, "a.go") != want {
		t.Errorf("write: exit %d, a.go %q, want %q:\n%s%s", wcode, subFile(t, root, "a.go"), want, wout, werr)
	}
	subTree(t, root, map[string]string{"a.go": a})
	if wout, werr, wcode := run("", "write", "--dry-run", "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", "."); wcode != 0 || !strings.Contains(wout, "2 hunk(s), 1 file(s), 0 failed") {
		t.Errorf("write --dry-run: exit %d:\n%s%s", wcode, wout, werr)
	}
	unchanged("a dry run")

	// nested matches: the outer and the inner foo of foo(foo(5, 6), 7)
	installRecordingAstGrep(t, rwStream(t, "a.go", a,
		[2]string{"foo(foo(5, 6), 7)", "bar(7, foo(5, 6))"},
		[2]string{"foo(5, 6)", "bar(6, 5)"}))
	out, errOut, code = run("", "read", "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", ".")
	if code != 1 || out != "" || !strings.Contains(errOut, "overlap") {
		t.Errorf("read of nested matches: exit %d, stdout %q, want 1, nothing printed and the overlap named:\n%s", code, out, errOut)
	}
	if _, errOut, code = run("", "write", "--no-check", "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", "."); code != 1 || !strings.Contains(errOut, "overlap") {
		t.Errorf("write of nested matches: exit %d, want 1 and the overlap named:\n%s", code, errOut)
	}
	unchanged("an overlap")

	// a match that changes no line is incomplete, and one that finds nothing is no match
	installRecordingAstGrep(t, rwStream(t, "a.go", a, [2]string{"foo(1, 2)", "foo(1, 2)"}))
	if _, errOut, code = run("", "read", "--ast-grep", "foo($A, $B)", "--rewrite", "foo($A, $B)", "."); code != 1 || !strings.Contains(errOut, "mrw: incomplete:") {
		t.Errorf("a rewrite that changes nothing: exit %d, want 1 incomplete:\n%s", code, errOut)
	}
	installRecordingAstGrep(t, "")
	if _, errOut, code = run("", "read", "--ast-grep", "nope", "--rewrite", "x", "."); code != 1 || !strings.Contains(errOut, "mrw: no match:") {
		t.Errorf("no match: exit %d, want 1 no match:\n%s", code, errOut)
	}
}

// ADR-161 Decision 3: --rewrite goes with --ast-grep, and with nothing that
// shapes served lines or makes a plan of its own.
func TestARewriteWithoutItsPartnersIsAUsageError(t *testing.T) {
	root, run := subRun(t)
	subTree(t, root, map[string]string{"a.go": "package a\n"})
	installRecordingAstGrep(t, "")
	for _, args := range [][]string{
		{"read", "--rewrite", "x", "."},
		{"read", "--grep", "a", "--rewrite", "x", "."},
		{"read", "--ast-grep", "p", "--rewrite", "x", "--sub", "y", "."},
		{"read", "--ast-grep", "p", "--rewrite", "x", "--count", "."},
		{"read", "--ast-grep", "p", "--rewrite", "x", "--stat", "."},
		{"read", "--ast-grep", "p", "--rewrite", "x", "--quiet", "."},
		{"read", "--ast-grep", "p", "--rewrite", "x", "--max-cols", "5", "."},
		{"read", "--ast-grep", "p", "--rewrite", "x", "--max-lines", "5", "."},
		{"read", "--ast-grep", "p", "--rewrite", "x", "-N", "."},
		{"read", "--ast-grep", "p", "--rewrite", "x", "a.go:1-2"},
		{"write", "--rewrite", "x", "."},
		{"write", "--ast-grep", "p", "."},
		{"write", "--ast-grep", "p", "--rewrite", "x", "-"},
		{"write", "--ast-grep", "p", "--rewrite", "x", "--format", "apply_patch", "."},
		{"write", "--ast-grep", "p", "--rewrite", "x", "--create", "n.txt", "."},
		{"write", "--ast-grep", "p", "--rewrite", "x", "--grep", "a", "--sub", "b", "."},
	} {
		if out, errOut, code := run("", args...); code != 2 {
			t.Errorf("%v: exit %d, want 2:\n%s%s", args, code, out, errOut)
		}
	}
}

// installScriptedAstGrep puts an ast-grep on PATH that answers its first
// start with out1 and every later one with out2, prints stderr text on each,
// and after its second start rewrites editPath to editBody (when editPath is
// set): a file changing between the two runs of a rewrite.
func installScriptedAstGrep(t *testing.T, out1, out2, stderr, editPath, editBody string) {
	t.Helper()
	dir := t.TempDir()
	count := filepath.Join(dir, "count")
	body := fmt.Sprintf(`package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 { // the untimed warm-up start
		return
	}
	f, _ := os.OpenFile(%q, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	_, _ = f.WriteString("x")
	_ = f.Close()
	b, _ := os.ReadFile(%q)
	out := %q
	if len(b) >= 2 {
		out = %q
	}
	fmt.Fprint(os.Stdout, out)
	fmt.Fprint(os.Stderr, %q)
	if len(b) >= 2 && %q != "" {
		_ = os.WriteFile(%q, []byte(%q), 0o644)
	}
}
`, count, count, out1, out2, stderr, editPath, editPath, editBody)
	src := filepath.Join(dir, "fake.go")
	if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "ast-grep")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	c := exec.Command("go", "build", "-o", bin, src)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("build fake ast-grep: %v\n%s", err, out)
	}
	_ = exec.Command(bin).Run() // the first start of a fresh binary is not timed
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// ADR-161 Decision 3: ast-grep 0.45.3 answers valid hits at exit 0 and names a
// path it could not read on stderr, so the scan is partial. A read prints the
// plan of what it found and exits 1 incomplete; a write applies nothing.
func TestARewriteOfAPartialScanIsIncompleteAndWritesNothing(t *testing.T) {
	const a = "package a\n\n\tfoo(1, 2)\n"
	root, run := subRun(t)
	subTree(t, root, map[string]string{"a.go": a})
	// both runs give the hit, so only the stderr line is under test
	hit := rwStream(t, "a.go", a, [2]string{"foo(1, 2)", "bar(2, 1)"})
	installScriptedAstGrep(t, hit, hit, "ERROR: d/bad: Permission denied (os error 13)\n", "", "")
	out, errOut, code := run("", "read", "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", ".")
	if code != 1 || !strings.Contains(out, "@@ a.go 3 replace") || !strings.Contains(errOut, "Permission denied") || !strings.Contains(errOut, "mrw: incomplete:") {
		t.Errorf("read: exit %d, want 1, the plan, the diagnostic named and incomplete last:\n%s%s", code, out, errOut)
	}
	if _, errOut, code = run("", "write", "--no-check", "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", "."); code != 1 || !strings.Contains(errOut, "Permission denied") {
		t.Errorf("write: exit %d, want 1 naming the diagnostic:\n%s", code, errOut)
	}
	if got := subFile(t, root, "a.go"); got != a {
		t.Errorf("a partial scan wrote a.go: %q", got)
	}
}

// ADR-161 Decision 3: the plan is bound to one version of each file. The file
// is read around a second ast-grep run over it and the two reads must agree,
// and the second run must find the same matches: a code line turned into a
// comment keeps the matched bytes and their offsets, which a check of the
// matched text alone does not see.
func TestARewriteIsRefusedWhenAFileChangesBetweenTwoRuns(t *testing.T) {
	const a = "package a\n\n    foo(1, 2)\n"
	const commented = "package a\n\n//  foo(1, 2)\n"
	hit := func() string { return rwStream(t, "a.go", a, [2]string{"foo(1, 2)", "bar(2, 1)"}) }
	for _, c := range []struct {
		name, out2, editBody, want string
	}{
		{"the file changes while the second run reads it", hit(), commented, "changed while ast-grep ran"},
		{"the second run finds no match in the changed file", "", "", "different matches"},
	} {
		t.Run(c.name, func(t *testing.T) {
			root, run := subRun(t)
			subTree(t, root, map[string]string{"a.go": a})
			edit := ""
			if c.editBody != "" {
				edit = filepath.Join(root, "a.go")
			}
			installScriptedAstGrep(t, hit(), c.out2, "", edit, c.editBody)
			if _, errOut, code := run("", "write", "--no-check", "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", "."); code != 1 || !strings.Contains(errOut, c.want) {
				t.Errorf("write: exit %d, want 1 naming %q:\n%s", code, c.want, errOut)
			}
			if got := subFile(t, root, "a.go"); got != a && got != commented {
				t.Errorf("a.go is %q, want it unwritten by the rewrite", got)
			}
		})
	}
}

// ADR-161 Decision 3: a file whose matches differ between the two ast-grep
// runs is named and refused, with the stable file beside it left unwritten
// too: the plan is all or nothing, and a differing set is not a stable one.
func TestARewriteWhoseSecondRunFindsOtherMatchesWritesNothing(t *testing.T) {
	const a = "package a\n\n\tfoo(1, 2)\n\tfoo(3, 4)\n"
	const c = "package c\n\tfoo(5, 6)\n"
	root, run := subRun(t)
	subTree(t, root, map[string]string{"a.go": a, "c.go": c})
	cs := rwStream(t, "c.go", c, [2]string{"foo(5, 6)", "bar(6, 5)"})
	both := rwStream(t, "a.go", a, [2]string{"foo(1, 2)", "bar(2, 1)"}, [2]string{"foo(3, 4)", "bar(4, 3)"}) + cs
	one := rwStream(t, "a.go", a, [2]string{"foo(1, 2)", "bar(2, 1)"}) + cs
	installScriptedAstGrep(t, both, one, "", "", "")
	if out, errOut, code := run("", "write", "--no-check", "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", "."); code != 1 || !strings.Contains(errOut, "different matches") {
		t.Errorf("write: exit %d, want 1 naming the different matches:\n%s%s", code, out, errOut)
	}
	if subFile(t, root, "a.go") != a || subFile(t, root, "c.go") != c {
		t.Errorf("a rewrite with a differing match set wrote a file")
	}
	installScriptedAstGrep(t, both, one, "", "", "")
	if _, errOut, code := run("", "read", "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", "."); code != 1 || !strings.Contains(errOut, "different matches") {
		t.Errorf("read: exit %d, want 1 naming the different matches:\n%s", code, errOut)
	}
}

// ADR-161 Decision 3: ast-grep 0.45.3 skips a file it cannot read without a
// word, exit 0 and an empty stderr. mrw walks the same paths itself, so an
// unreadable file is named and the rewrite writes nothing.
func TestARewriteBesideAnUnreadableFileWritesNothing(t *testing.T) {
	const good = "package a\n\n\tfoo(1, 2)\n"
	root, run := subRun(t)
	subTree(t, root, map[string]string{"good.go": good, "bad.go": "package a\n\n\tfoo(3, 4)\n"})
	bad := filepath.Join(root, "bad.go")
	if err := os.Chmod(bad, 0); err != nil {
		t.Skipf("permission bits cannot be set here: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(bad, 0o644) })
	if f, err := os.Open(bad); err == nil {
		_ = f.Close()
		t.Skip("permission bits are not enforced here (running as root?)")
	}
	hit := rwStream(t, "good.go", good, [2]string{"foo(1, 2)", "bar(2, 1)"})
	installScriptedAstGrep(t, hit, hit, "", "", "")
	out, errOut, code := run("", "write", "--no-check", "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", ".")
	if code != 1 || !strings.Contains(errOut, "bad.go") || !strings.Contains(errOut, "cannot be scanned") {
		t.Errorf("write: exit %d, want 1 naming bad.go:\n%s%s", code, out, errOut)
	}
	if got := subFile(t, root, "good.go"); got != good {
		t.Errorf("a rewrite beside an unreadable file wrote good.go: %q", got)
	}
}
