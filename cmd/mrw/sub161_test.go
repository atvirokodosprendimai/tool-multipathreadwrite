package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// subRun builds mrw once for a test and returns a runner over one root and one
// state directory: stdout, stderr and the exit code, never through a pipe.
func subRun(t *testing.T) (root string, run func(stdin string, args ...string) (string, string, int)) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "mrw.exe") // .exe: Windows exec will not launch an extensionless file
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	root, state := t.TempDir(), t.TempDir()
	return root, func(stdin string, args ...string) (string, string, int) {
		c := exec.Command(bin, append([]string{"--root", root}, args...)...)
		c.Env = append(os.Environ(), "XDG_STATE_HOME="+state)
		c.Stdin = strings.NewReader(stdin)
		var o, e strings.Builder
		c.Stdout, c.Stderr = &o, &e
		err := c.Run()
		if ee, ok := err.(*exec.ExitError); ok {
			return o.String(), e.String(), ee.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		return o.String(), e.String(), 0
	}
}

func subTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func subFile(t *testing.T, root, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// ADR-161 Decision 2. `write --grep P --sub R` builds the read's plan and
// applies it as one plan: a verdict per site in one receipt, nothing written
// when any site fails or any file it would have planned could not be read,
// --dry-run writes nothing, a CRLF file keeps its endings, and the write's
// own steps run.
func TestASubWriteAppliesThePlanInOneCommand(t *testing.T) {
	root, run := subRun(t)
	files := map[string]string{
		"a.go":     "Old\nOldX\nxOld\nkeep Old and Old\n",
		"sub/c.go": "Old\n",
		"crlf.txt": "Old\r\nkeep\r\n",
	}
	receipt := "4 hunk(s), 3 file(s), 0 failed"
	if runtime.GOOS != "windows" { // a name the plan header must quote and escape
		files[`q"uote and 'tick'.txt`] = "Old\n"
		receipt = "5 hunk(s), 4 file(s), 0 failed"
	}
	subTree(t, root, files)
	restore := func() { subTree(t, root, files) }
	unchanged := func(why string) {
		t.Helper()
		for name, body := range files {
			if got := subFile(t, root, name); got != body {
				t.Errorf("%s: %s changed %s to %q", why, why, name, got)
			}
		}
	}

	// A dry run reports every site and writes nothing.
	out, errOut, code := run("", "write", "--dry-run", "--grep", `\bOld\b`, "--sub", "New", ".")
	if code != 0 || !strings.Contains(out, receipt) {
		t.Errorf("--dry-run: exit %d, want 0 and %s:\n%s%s", code, receipt, out, errOut)
	}
	unchanged("--dry-run")

	// The real write: one receipt, every site changed, no neighbour, CRLF kept,
	// and a --then-sh step runs after it.
	out, errOut, code = run("", "write", "--grep", `\bOld\b`, "--sub", "New", "--then-sh", "echo ran > then.ran", ".")
	if code != 0 || !strings.Contains(out, receipt) {
		t.Fatalf("write --sub: exit %d, want 0 and one receipt of %s:\n%s%s", code, receipt, out, errOut)
	}
	if got := subFile(t, root, "a.go"); got != "New\nOldX\nxOld\nkeep New and New\n" {
		t.Errorf("a.go = %q", got)
	}
	if got := subFile(t, root, "sub/c.go"); got != "New\n" {
		t.Errorf("sub/c.go = %q", got)
	}
	if got := subFile(t, root, "crlf.txt"); got != "New\r\nkeep\r\n" {
		t.Errorf("crlf.txt = %q, want its CRLF endings kept", got)
	}
	if runtime.GOOS != "windows" {
		if got := subFile(t, root, `q"uote and 'tick'.txt`); got != "New\n" {
			t.Errorf("the name with both quote kinds = %q, want the plan to have carried it", got)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "then.ran")); err != nil {
		t.Errorf("--then-sh did not run: %v", err)
	}
	restore()

	// A site that fails fails the plan: a read-only file is refused by the
	// engine, and no sibling is written.
	if runtime.GOOS != "windows" {
		ro := filepath.Join(root, "sub", "c.go")
		if err := os.Chmod(ro, 0o444); err != nil {
			t.Fatal(err)
		}
		out, errOut, code = run("", "write", "--grep", `\bOld\b`, "--sub", "New", ".")
		if code != 1 || !strings.Contains(out, "c.go") {
			t.Errorf("a read-only site: exit %d, want 1 naming the file:\n%s%s", code, out, errOut)
		}
		unchanged("a failed site")
		_ = os.Chmod(ro, 0o644)

		// A matching file the walk could not read: nothing is written, exit 1,
		// the file named. Skipped as root, which reads anything.
		if os.Geteuid() != 0 {
			if err := os.Chmod(ro, 0); err != nil {
				t.Fatal(err)
			}
			out, errOut, code = run("", "write", "--grep", `\bOld\b`, "--sub", "New", ".")
			_ = os.Chmod(ro, 0o644)
			if code != 1 || !strings.Contains(out+errOut, "c.go") {
				t.Errorf("an unreadable matching file: exit %d, want 1 naming it:\n%s%s", code, out, errOut)
			}
			unchanged("an unreadable file")
		}
	}

	// Nothing planned writes nothing: a substitution that changes no line is
	// `incomplete:`, a pattern that matches no file is `no match:`, both exit 1.
	out, errOut, code = run("", "write", "--grep", "Old", "--sub", "Old", ".")
	if code != 1 || !strings.Contains(errOut, "mrw: incomplete:") || out != "" {
		t.Errorf("a sub that changes nothing: exit %d, want 1 incomplete:\n%s%s", code, out, errOut)
	}
	_, errOut, code = run("", "write", "--grep", "NoSuchText", "--sub", "x", ".")
	if code != 1 || !strings.Contains(errOut, "mrw: no match:") {
		t.Errorf("no file matched: exit %d, want 1 no match:\n%s", code, errOut)
	}
	unchanged("an empty plan")

	// --exclude and --no-ignore shape the walk as read --grep's do: an
	// excluded file is not planned. Under --json a refusal is one document
	// that keeps its exit (ADR-072).
	out, errOut, code = run("", "write", "--dry-run", "--grep", `\bOld\b`, "--sub", "New", "--exclude", "sub", ".")
	if code != 0 || strings.Contains(out, "c.go") || !strings.Contains(out, "a.go") {
		t.Errorf("--exclude sub: exit %d, want 0 and sub/c.go left out:\n%s%s", code, out, errOut)
	}
	out, errOut, code = run("", "write", "--json", "--grep", "NoSuchText", "--sub", "x", ".")
	if code != 1 || !strings.HasPrefix(strings.TrimSpace(out), "{") || !strings.Contains(out, `"error": "no match:`) {
		t.Errorf("--json no match: exit %d, want 1 and one JSON document:\n%s%s", code, out, errOut)
	}
	if _, errOut, code := run("", "write", "--exclude", "x", "plan.txt"); code != 2 {
		t.Errorf("--exclude without --grep: exit %d, want 2:\n%s", code, errOut)
	}

	// No PLAN beside it, and nothing --sub cannot be combined with.
	for _, args := range [][]string{
		{"write", "--grep", "Old", "--sub", "New", "-"},
		{"write", "--grep", "Old", "--sub", "New", "--format", "apply_patch", "."},
		{"write", "--grep", "Old", "."},
		{"write", "--sub", "New", "."},
	} {
		if _, errOut, code := run("", args...); code != 2 {
			t.Errorf("mrw %s: exit %d, want 2:\n%s", strings.Join(args, " "), code, errOut)
		}
	}
	unchanged("a refusal")
}

// ADR-161 Decision 1, through the binary: the plan is all of stdout, the count
// and the footer are stderr, the licence is the planned lines, and the usage
// refusals are exit 2 before anything is read.
func TestASubReadRefusalsAndExits(t *testing.T) {
	root, run := subRun(t)
	subTree(t, root, map[string]string{"a.go": "Old\nOldX\nxOld\nOld Old\n", "b.go": "Old\n", "bin.dat": "Old\x00 binary\n"})

	out, errOut, code := run("", "read", "--grep", `\bOld\b`, "--sub", "New", ".")
	if code != 0 || strings.Count(out, " replace sha=") != 3 || !strings.Contains(out, " body=1 raw=true\nNew New\n") || !strings.HasPrefix(out, "@@ a.go 1 replace sha=") {
		t.Errorf("read --sub: exit %d, stdout %q, want 0 and three sha-pinned raw hunks", code, out)
	}
	if !strings.Contains(errOut, "1 binary file(s) that matched") {
		t.Errorf("the binary file the walk skipped is not counted on stderr:\n%s", errOut)
	}
	if !strings.Contains(errOut, "-- planned: 3 site(s) in 2 file(s)") {
		t.Errorf("stderr lacks the planned line:\n%s", errOut)
	}
	// Licensed and not shown (ADR-118): occurrence=N counts only lines a read
	// showed, and this read showed none of the planned ones.
	// The pattern matches lines 1 and 4, both planned, so occurrence=2 is licensed
	// itself and only the unshown line 1 before it can refuse it.
	if o, e, c := run("@@ a.go /^(Old|Old Old)$/ replace occurrence=2\nzzz\n", "write", "-"); c == 0 || subFile(t, root, "a.go") != "Old\nOldX\nxOld\nOld Old\n" {
		t.Errorf("occurrence=2 over a planned-but-unshown line 1: exit %d, want a refusal:\n%s%s", c, o, e)
	} else if !strings.Contains(o+e, "a --sub plan licensed line") {
		t.Errorf("the refusal does not say the lines were not shown:\n%s%s", o, e)
	}
	// Lines it did not plan are not licensed: line 2 was served by no read.
	if o, _, c := run("@@ a.go 2 replace\nzzz\n", "write", "-"); c == 0 {
		t.Errorf("an unplanned line was writable:\n%s", o)
	}
	// The plan's lines are: it applies without a further read.
	if o, e, c := run(out, "write", "-"); c != 0 {
		t.Errorf("write - of the sub plan: exit %d:\n%s%s", c, o, e)
	}

	for _, args := range [][]string{
		{"--sub", "x", "."},
		{"--grep", "Old", "--sub", "x", "--ast-grep", "P", "."},
		{"--grep", "Old", "--sub", "x", "-C", "1", "."},
		{"--grep", "Old", "--sub", "x", "--count", "."},
		{"--grep", "Old", "--sub", "x", "--stat", "."},
		{"--grep", "Old", "--sub", "x", "--max-cols", "5", "."},
		{"--grep", "Old", "--sub", "x", "--max-lines", "5", "."},
		{"--grep", "Old", "--sub", "x", "--no-numbers", "."},
		{"--grep", "Old", "--sub", "x", "--files-from", "-"},
		{"--grep", "Old", "--sub", "a\nb", "."},
		{"--grep", "Old", "--sub", "x", "a.go:1-2"},
	} {
		if o, e, c := run("", append([]string{"read"}, args...)...); c != 2 || o != "" {
			t.Errorf("mrw read %s: exit %d stdout %q, want 2 and nothing:\n%s", strings.Join(args, " "), c, o, e)
		}
	}
	if _, e, c := run("", "read", "--grep", "Old", "--sub", "Old", "."); c != 1 || !strings.Contains(e, "mrw: incomplete:") {
		t.Errorf("a sub that changes nothing: exit %d, want 1 incomplete:\n%s", c, e)
	}
	if _, e, c := run("", "read", "--grep", "NoSuchText", "--sub", "x", "."); c != 1 || !strings.Contains(e, "mrw: no match:") {
		t.Errorf("no file matched: exit %d, want 1 no match:\n%s", c, e)
	}
}

// A file whose name starts with @ is planned as ./@…, never read back as a
// pointer into the working set: with entry 1 on another file holding the same
// text, licensed and excluded, the write changes @1 and leaves the other alone
// (Codex review of ADR-161 T1).
func TestASubWriteNeverReadsAFileNameAsAPointer(t *testing.T) {
	root, run := subRun(t)
	subTree(t, root, map[string]string{"@1": "Old\n", "target.txt": "Old\n"})
	if _, e, c := run("", "read", "target.txt"); c != 0 {
		t.Fatalf("read target.txt: exit %d\n%s", c, e)
	}
	if _, e, c := run("", "iter", "add", "target.txt"); c != 0 {
		t.Fatalf("iter add: exit %d\n%s", c, e)
	}
	out, errOut, code := run("", "write", "--no-check", "--grep", `\bOld\b`, "--sub", "New", "--exclude", "target.txt", ".")
	if code != 0 || subFile(t, root, "@1") != "New\n" || subFile(t, root, "target.txt") != "Old\n" {
		t.Errorf("exit %d, @1 %q, target.txt %q; want 0, @1 changed and target.txt not:\n%s%s",
			code, subFile(t, root, "@1"), subFile(t, root, "target.txt"), out, errOut)
	}
}
