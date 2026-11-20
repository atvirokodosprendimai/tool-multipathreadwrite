package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-208 T1, through the built command: a multi-line replace on a file mrw wrote
// (or read whole with --quiet) needs the line after the range shown since the
// write. f.go is func A() { / if x { / } / }, and the plan replaces 1-3.
const (
	fGo208  = "func A() {\n\tif x {\n\t}\n}\n"
	rep208  = "@@ f.go 1-3 replace anchor=\"func A\"\nfunc A() {\n\ty()\n}\n"
	a264Go  = "package a\n\nfunc f() {\n\tfoo(1, 2)\n\tx := foo(\n\t\t3,\n\t\t4)\n\t_ = foo(foo(5, 6), 7)\n\ts := \"naïve\"; foo(s, \"ŝ\")\n\t_ = x\n}\n"
	a264Out = "package a\n\nfunc f() {\n\tbar(2, 1)\n\tx := bar(4, 3)\n\t_ = foo(foo(5, 6), 7)\n\ts := \"naïve\"; bar(\"ŝ\", s)\n\t_ = x\n}\n"
	// stream264 is what ast-grep 0.45.3 printed for `--pattern 'foo($A, $B)'
	// --rewrite 'bar($B, $A)' --json=stream` on a264Go (contract §264).
	stream264 = `{"text":"foo(1, 2)","range":{"byteOffset":{"start":23,"end":32},"start":{"line":3,"column":1},"end":{"line":3,"column":10}},"file":"a.go","lines":"\tfoo(1, 2)","charCount":{"leading":1,"trailing":0},"replacement":"bar(2, 1)","replacementOffsets":{"start":23,"end":32},"language":"Go","metaVariables":{"single":{},"multi":{},"transformed":{}}}
{"text":"foo(\n\t\t3,\n\t\t4)","range":{"byteOffset":{"start":39,"end":53},"start":{"line":4,"column":6},"end":{"line":6,"column":4}},"file":"a.go","lines":"\tx := foo(\n\t\t3,\n\t\t4)","charCount":{"leading":6,"trailing":0},"replacement":"bar(4, 3)","replacementOffsets":{"start":39,"end":53},"language":"Go","metaVariables":{"single":{},"multi":{},"transformed":{}}}
{"text":"foo(s, \"ŝ\")","range":{"byteOffset":{"start":93,"end":105},"start":{"line":8,"column":15},"end":{"line":8,"column":26}},"file":"a.go","lines":"\ts := \"naïve\"; foo(s, \"ŝ\")","charCount":{"leading":15,"trailing":0},"replacement":"bar(\"ŝ\", s)","replacementOffsets":{"start":93,"end":105},"language":"Go","metaVariables":{"single":{},"multi":{},"transformed":{}}}
`
)

func TestTheCLIRefusesAMultiLineReplaceOnAWrittenFileUntilTheLineAfterIsShown(t *testing.T) {
	needShell(t)
	create := func(t *testing.T, root, name, body string) {
		t.Helper()
		if out, code := writeIn(t, root, "--no-check", planFile(t, "@@ "+name+" 0 create\n"+strings.TrimSuffix(body, "\n")+"\n")); code != 0 {
			t.Fatalf("create %s: exit %d\n%s", name, code, out)
		}
	}
	slurp := func(t *testing.T, root, name string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	t.Run("numbered", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := t.TempDir()
		create(t, root, "f.go", fGo208)
		out, code := writeIn(t, root, "--no-check", "--json", planFile(t, rep208))
		if code != 1 || !strings.Contains(out, "needs a served line after 3") || !strings.Contains(out, `"failed"`) {
			t.Fatalf("exit %d, want 1 with the failed hunk:\n%s", code, out)
		}
		if got := slurp(t, root, "f.go"); got != fGo208 {
			t.Errorf("a refused plan changed f.go: %q", got)
		}
		if out, code := runIn(t, root, "read", "f.go:4"); code != 0 {
			t.Fatalf("read: %d\n%s", code, out)
		}
		if out, code := writeIn(t, root, "--no-check", planFile(t, rep208)); code != 0 {
			t.Fatalf("after the read of line 4: exit %d\n%s", code, out)
		}
	})
	t.Run("whole_quiet_read", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "f.go"), []byte(fGo208), 0o644); err != nil {
			t.Fatal(err)
		}
		if out, code := runIn(t, root, "read", "--quiet", "f.go"); code != 0 {
			t.Fatalf("quiet read: %d\n%s", code, out)
		}
		if out, code := writeIn(t, root, "--no-check", planFile(t, rep208)); code != 1 || !strings.Contains(out, "needs a served line after 3") {
			t.Errorf("after a whole quiet read: exit %d, want 1:\n%s", code, out)
		}
	})
	t.Run("ranged_quiet_read", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := t.TempDir()
		create(t, root, "f.go", fGo208)
		if out, code := runIn(t, root, "read", "--quiet", "f.go:1-3"); code != 0 {
			t.Fatalf("ranged quiet read: %d\n%s", code, out)
		}
		if out, code := writeIn(t, root, "--no-check", planFile(t, rep208)); code != 1 || !strings.Contains(out, "needs a served line after 3") {
			t.Errorf("after a ranged quiet read: exit %d, want 1:\n%s", code, out)
		}
	})
	t.Run("same_call_rewrite", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := t.TempDir()
		create(t, root, "a.go", a264Go)
		installFakeAstGrep(t, stream264, 0)
		if out, code := writeIn(t, root, "--no-check", "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", "."); code != 0 {
			t.Fatalf("a same-call rewrite: exit %d\n%s", code, out)
		}
		if got := slurp(t, root, "a.go"); got != a264Out {
			t.Errorf("a.go is %q, want the rewrite", got)
		}
	})
	t.Run("later_call_rewrite", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := t.TempDir()
		create(t, root, "a.go", a264Go)
		installFakeAstGrep(t, stream264, 0)
		planOut, code := runIn(t, root, "read", "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", ".")
		if code != 0 {
			t.Fatalf("read --rewrite: %d\n%s", code, planOut)
		}
		plan := filterPlan208(planOut)
		out, code := writeIn(t, root, "--no-check", planFile(t, plan))
		if code != 1 || !strings.Contains(out, "needs a served line after 7") {
			t.Fatalf("the plan written later: exit %d, want 1 naming the line after:\n%s", code, out)
		}
		if got := slurp(t, root, "a.go"); got != a264Go {
			t.Errorf("a refused plan changed a.go: %q", got)
		}
		if out, code := runIn(t, root, "read", "a.go:8"); code != 0 {
			t.Fatalf("read: %d\n%s", code, out)
		}
		if out, code := writeIn(t, root, "--no-check", planFile(t, plan)); code != 0 {
			t.Fatalf("after the read of line 8: exit %d\n%s", code, out)
		}
		if got := slurp(t, root, "a.go"); got != a264Out {
			t.Errorf("a.go is %q, want the rewrite", got)
		}
	})
}

// filterPlan208 keeps what `mrw read --rewrite` printed as the plan: the @@ lines
// and their bodies, without the "-- planned" notes.
func filterPlan208(out string) string {
	var b strings.Builder
	for _, l := range strings.SplitAfter(out, "\n") {
		if strings.HasPrefix(l, "-- ") || strings.HasPrefix(l, "mrw:") {
			continue
		}
		b.WriteString(l)
	}
	return b.String()
}
