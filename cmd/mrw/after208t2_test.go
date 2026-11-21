package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-208 Decision 1a through the built command. f.go is the record's repro: five
// lines, then a search_replace block that replaces `var z = 1` with five lines, so
// lines 5-9 are written by the caller's earlier call and licensed, not shown; the
// plan replaces 6-8 with a body ending in a closer and leaves line 9's behind.
func TestTheCLIRefusesAReplaceEndingAboveLinesASearchReplaceWrote(t *testing.T) {
	needShell(t)
	const (
		five    = "package a\n\nfunc A() {}\n\nvar z = 1\n"
		block   = "f.go\n<<<<<<< SEARCH\nvar z = 1\n=======\nvar z = 2\nfunc B() {\n\tif y {\n\t}\n}\n>>>>>>> REPLACE\n"
		replace = "@@ f.go 6-8 replace anchor=\"func B\"\nfunc B() {\n\tz()\n}\n"
		written = "package a\n\nfunc A() {}\n\nvar z = 2\nfunc B() {\n\tif y {\n\t}\n}\n"
	)
	// fixture is a root holding f.go after the search_replace write.
	fixture := func(t *testing.T) string {
		t.Helper()
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := t.TempDir()
		if out, code := writeIn(t, root, "--no-check", planFile(t, "@@ f.go 0 create\n"+strings.TrimSuffix(five, "\n")+"\n")); code != 0 {
			t.Fatalf("create: %d\n%s", code, out)
		}
		if out, code := writeIn(t, root, "--no-check", "--format=search_replace", planFile(t, block)); code != 0 {
			t.Fatalf("search_replace: %d\n%s", code, out)
		}
		if got, _ := os.ReadFile(filepath.Join(root, "f.go")); string(got) != written {
			t.Fatalf("f.go is %q after the write", got)
		}
		return root
	}
	unchanged := func(t *testing.T, root, name string) {
		t.Helper()
		if got, _ := os.ReadFile(filepath.Join(root, name)); string(got) != written {
			t.Errorf("a refused plan changed %s: %q", name, got)
		}
	}
	t.Run("refused_then_read", func(t *testing.T) {
		root := fixture(t)
		out, code := writeIn(t, root, "--no-check", planFile(t, replace))
		if code != 1 || !strings.Contains(out, "needs a served line after 8: read line 9") {
			t.Fatalf("exit %d, want 1 naming line 9:\n%s", code, out)
		}
		unchanged(t, root, "f.go")
		if rout, rcode := runIn(t, root, remedyArgv(t, root, out, "f.go:6-9")...); rcode != 0 {
			t.Fatalf("the remedy read: %d\n%s", rcode, rout)
		}
		if out, code := writeIn(t, root, "--no-check", planFile(t, replace)); code != 0 {
			t.Errorf("after the read: exit %d\n%s", code, out)
		}
	})
	t.Run("seen_says_licensed", func(t *testing.T) {
		root := fixture(t)
		out, code := runIn(t, root, "seen")
		if code != 0 || !strings.Contains(out, "f.go") || !strings.Contains(out, "no lines, and licensed lines 5-9") {
			t.Errorf("mrw seen: exit %d, want f.go licensed 5-9 and not shown:\n%s", code, out)
		}
	})
	t.Run("alias_quiet_then_read", func(t *testing.T) {
		root := fixture(t)
		if err := os.Symlink("f.go", filepath.Join(root, "g.go")); err != nil {
			t.Skipf("no symlinks: %v", err)
		}
		plan := strings.Replace(replace, "@@ f.go", "@@ g.go", 1)
		if out, code := runIn(t, root, "read", "--quiet", "g.go:6-9"); code != 0 {
			t.Fatalf("quiet read: %d\n%s", code, out)
		}
		out, code := writeIn(t, root, "--no-check", planFile(t, plan))
		if code != 1 || !strings.Contains(out, "needs a served line after 8") {
			t.Fatalf("exit %d, want 1 with the line-after reason:\n%s", code, out)
		}
		unchanged(t, root, "f.go")
		if rout, rcode := runIn(t, root, remedyArgv(t, root, out, "g.go:6-9")...); rcode != 0 {
			t.Fatalf("the remedy read: %d\n%s", rcode, rout)
		}
		if out, code := writeIn(t, root, "--no-check", planFile(t, plan)); code != 0 {
			t.Errorf("after the read: exit %d\n%s", code, out)
		}
	})
	t.Run("from_copy_needs_a_read", func(t *testing.T) {
		root := fixture(t)
		copyPlan := planFile(t, "@@ h.go 0 create from=f.go:6\n")
		if out, code := writeIn(t, root, "--no-check", copyPlan); code != 1 {
			t.Fatalf("a copy of lines no read showed: exit %d, want 1\n%s", code, out)
		}
		if _, err := os.Stat(filepath.Join(root, "h.go")); err == nil {
			t.Error("a refused copy made h.go")
		}
		if out, code := runIn(t, root, "read", "f.go:6"); code != 0 {
			t.Fatalf("read: %d\n%s", code, out)
		}
		if out, code := writeIn(t, root, "--no-check", copyPlan); code != 0 {
			t.Errorf("after the read: exit %d\n%s", code, out)
		}
	})
	t.Run("if_served_serves_written_lines", func(t *testing.T) {
		root := fixture(t)
		sha := seen.SHA([]byte(written))[:8]
		out, code := runIn(t, root, "read", "--if-served", "f.go="+sha, "f.go:5-9")
		if code != 0 || strings.Contains(out, "-- same as served") || !strings.Contains(out, "func B() {") {
			t.Errorf("--if-served over written lines: exit %d, want the lines served, not a same-as-served line:\n%s", code, out)
		}
	})
	// ADR-118: occurrence=N counts matches in lines the caller saw. After a real search_replace write the
	// lines it produced are licensed and not shown, so a match among them is unread (they were read-shaped before).
	t.Run("occurrence_after_a_written_match", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := t.TempDir()
		if out, code := writeIn(t, root, "--no-check", planFile(t, "@@ f.go 0 create\n"+strings.TrimSuffix(five, "\n")+"\n")); code != 0 {
			t.Fatalf("create: %d\n%s", code, out)
		}
		twice := "f.go\n<<<<<<< SEARCH\nvar z = 1\n=======\nvar z = 2\n\t}\nfunc B() {\n\t}\n}\n>>>>>>> REPLACE\n"
		if out, code := writeIn(t, root, "--no-check", "--format=search_replace", planFile(t, twice)); code != 0 {
			t.Fatalf("search_replace: %d\n%s", code, out)
		}
		plan := planFile(t, "@@ f.go /^\\t}/ replace anchor=\"}\" occurrence=2\n\t} // x\n")
		out, code := writeIn(t, root, "--no-check", plan)
		if code != 1 || !strings.Contains(out, "have not been read") {
			t.Fatalf("exit %d, want 1 naming the unread earlier match:\n%s", code, out)
		}
		if rout, rcode := runIn(t, root, "read", "f.go:6-7"); rcode != 0 {
			t.Fatalf("read: %d\n%s", rcode, rout)
		}
		if out, code := writeIn(t, root, "--no-check", plan); code != 0 {
			t.Errorf("after reading the earlier match: exit %d\n%s", code, out)
		}
	})
	t.Run("subroot_remedy_from_the_original_directory", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		top := t.TempDir()
		root := filepath.Join(top, "sub")
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		// a git checkout holding sub/f.go: the ledger is the checkout's, the root is below it
		if err := os.MkdirAll(filepath.Join(top, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		if out, code := writeIn(t, root, "--no-check", planFile(t, "@@ f.go 0 create\n"+strings.TrimSuffix(five, "\n")+"\n")); code != 0 {
			t.Fatalf("create: %d\n%s", code, out)
		}
		if out, code := writeIn(t, root, "--no-check", "--format=search_replace", planFile(t, block)); code != 0 {
			t.Fatalf("search_replace: %d\n%s", code, out)
		}
		out, code := writeIn(t, root, "--no-check", planFile(t, replace))
		if code != 1 || !strings.Contains(out, "needs a served line after 8") {
			t.Fatalf("exit %d, want 1 with the line-after reason:\n%s", code, out)
		}
		// the remedy names the root, and run as printed it licenses the retry
		if plainPath.MatchString(root) && !strings.Contains(out, "mrw --root "+shellRoot(root)+" read --max-cols 0 f.go:6-9") {
			t.Errorf("the remedy does not name the root %s:\n%s", root, out)
		}
		if rout, rcode := runIn(t, root, remedyArgv(t, root, out, "f.go:6-9")...); rcode != 0 {
			t.Fatalf("the remedy read: %d\n%s", rcode, rout)
		}
		if out, code := writeIn(t, root, "--no-check", planFile(t, replace)); code != 0 {
			t.Errorf("after the read: exit %d\n%s", code, out)
		}
	})
	t.Run("saved_rewrite_plan", func(t *testing.T) {
		const a = "package a\n\nfunc f() {\n\tfoo(1, 2)\n\tx := foo(\n\t\t3,\n\t\t4)\n\t_ = foo(foo(5, 6), 7)\n\ts := \"naïve\"; foo(s, \"ŝ\")\n\t_ = x\n}\n"
		const want = "package a\n\nfunc f() {\n\tbar(2, 1)\n\tx := bar(4, 3)\n\t_ = foo(foo(5, 6), 7)\n\ts := \"naïve\"; bar(\"ŝ\", s)\n\t_ = x\n}\n"
		root, run := subRun(t)
		subTree(t, root, map[string]string{"a.go": a})
		installRecordingAstGrep(t, rwStream(t, "a.go", a,
			[2]string{"foo(1, 2)", "bar(2, 1)"},
			[2]string{"foo(\n\t\t3,\n\t\t4)", "bar(4, 3)"},
			[2]string{`foo(s, "ŝ")`, `bar("ŝ", s)`}))
		plan, _, code := run("", "read", "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", ".")
		if code != 0 {
			t.Fatalf("read --rewrite: exit %d", code)
		}
		wout, werr, wcode := run(plan, "write", "--no-check", "-")
		if wcode != 1 || !strings.Contains(wout+werr, "needs a served line after 7") || subFile(t, root, "a.go") != a {
			t.Fatalf("the saved plan, written by a later call: exit %d, a.go %q:\n%s%s", wcode, subFile(t, root, "a.go"), wout, werr)
		}
		if rout, rerr, rcode := run("", "read", "a.go:8"); rcode != 0 {
			t.Fatalf("read: %d\n%s%s", rcode, rout, rerr)
		}
		if wout, werr, wcode := run(plan, "write", "--no-check", "-"); wcode != 0 || subFile(t, root, "a.go") != want {
			t.Errorf("after the read: exit %d, a.go %q\n%s%s", wcode, subFile(t, root, "a.go"), wout, werr)
		}
		// the same-call rewrite, on a fresh copy, is not judged
		subTree(t, root, map[string]string{"a.go": a})
		if wout, werr, wcode := run("", "write", "--no-check", "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", "."); wcode != 0 || subFile(t, root, "a.go") != want {
			t.Errorf("the same-call rewrite: exit %d, a.go %q\n%s%s", wcode, subFile(t, root, "a.go"), wout, werr)
		}
	})
}
