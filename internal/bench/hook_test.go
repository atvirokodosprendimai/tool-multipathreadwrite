package bench

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

func hookInput(t *testing.T, tool, command string) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"hook_event_name": "PreToolUse",
		"tool_name":       tool,
		"tool_input":      map[string]any{"command": command},
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestTheArmBHookRefusesAWriteByAnyMeansButMrwAndCountsReads drives the hook as Claude
// Code would, with a PreToolUse event on stdin. Writes by any means but mrw are
// refused (exit 2, the reason on stderr); reads through cat, grep, head, sed -n
// and their kin pass and are counted, not refused (Zy may overrule that
// default, ADR-156 Decision 1); everything else passes uncounted.
func TestTheArmBHookRefusesAWriteByAnyMeansButMrwAndCountsReads(t *testing.T) {
	refuse := []string{
		"sed -i 's/a/b/' f.go",
		"echo x > f.txt",
		"cat a.txt >> b.txt",
		"printf x | tee f.txt",
		"cp a.go b.go",
		"mv a.go b.go",
		"rm f.go",
		"touch new.go",
		"perl -pi -e 's/a/b/' f.go",
		"python3 -c \"open('f','w').write('x')\"",
		"node -e \"require('fs').writeFileSync('f','x')\"",
		"git apply fix.patch",
		"git checkout -- f.go",
		"patch -p1 < fix.patch",
		"dd if=a of=b",
		"truncate -s 0 f.go",
		"go fmt ./...",
		"gofmt -w f.go",
		"cat <<'EOF' > f.go\npackage x\nEOF",
		"ls && echo hi > out.txt",
		// What the review of #411 found passing:
		"echo \"$(touch f)\"",
		"cat <<EOF\n$(rm f)\nEOF",
		"echo \"<<EOF\"; rm f",
		"printf x >&out.txt",
		"echo `touch f`",
		// And what its re-review found:
		"echo \"'$(touch f)'\"",
		"echo \"'\"; touch f; echo \"'\"",
		"env cp a.go b.go",
		"FOO=1 rm f.go",
		"timeout 5 mv a b",
		"command touch f",
		// And its second re-review: a substitution with quotes of its own.
		"echo \"$(printf \"'\"; touch f; printf \"'\")\"",
		"x=$(cat a.txt) && rm $x",
		// And its third: arithmetic, and quotes a heredoc keeps as text.
		"echo $(( $(touch f) + 1 ))",
		"cat <<EOF\n\" '$(touch f)' \"\nEOF",
		// And its fourth: arithmetic expands as double quotes do.
		"echo $(( '$(touch f)' + 1 ))",
	}
	pass := []string{
		"mrw read f.go:1-3",
		"mrw write - <<'PLAN'\n@@ f.go 1 replace\nx := a > b\nPLAN",
		"go test ./... 2>&1",
		"go vet ./...",
		"git status",
		"git diff",
		"ls -la > /dev/null",
		"mrw read f.go 2>/dev/null",
		"go build ./... 2>&1 >/dev/null",
		"echo hi >&2",
		"ls 2>&1",
		"echo $((1 + 2))",
		"command -v cp",
		"echo \"$(mrw read f.go:1-2)\"",
		"mrw write - <<'PLAN'\n@@ f.go 1 replace\nv := \"$(touch f)\"\nPLAN",
	}
	reads := map[string][]string{
		"cat f.go":               {"cat"},
		"grep -n x f.go":         {"grep"},
		"head -5 f.go":           {"head"},
		"sed -n 1,5p f.go":       {"sed"},
		"mrw read f.go | grep x": nil,
		"rg -l x .":              {"rg"},
	}
	run := func(cmd string) (int, string, string) {
		log := filepath.Join(t.TempDir(), "hook.log")
		var stderr bytes.Buffer
		code := Hook(bytes.NewReader(hookInput(t, "Bash", cmd)), &stderr, log, "")
		b, _ := os.ReadFile(log)
		return code, stderr.String(), string(b)
	}
	for _, c := range refuse {
		if code, stderr, log := run(c); code != 2 || stderr == "" || !strings.Contains(log, "refused\t") {
			t.Errorf("not refused: %q (exit %d, stderr %q, log %q)", c, code, stderr, log)
		}
	}
	for _, c := range pass {
		if code, stderr, log := run(c); code != 0 || strings.Contains(log, "refused") {
			t.Errorf("refused: %q (exit %d, stderr %q)", c, code, stderr)
		}
	}
	for c, want := range reads {
		code, _, log := run(c)
		var got []string
		for _, l := range strings.Split(strings.TrimSpace(log), "\n") {
			if tool, ok := strings.CutPrefix(l, "read\t"); ok {
				got = append(got, tool)
			}
		}
		if code != 0 || strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%q: exit %d, counted %v, want %v", c, code, got, want)
		}
	}
	if code := Hook(bytes.NewReader(hookInput(t, "Glob", "")), new(bytes.Buffer), filepath.Join(t.TempDir(), "l"), ""); code != 0 {
		t.Errorf("a tool other than Bash was refused: exit %d", code)
	}
}

// TestAWriteOutsideTheTaskTreePassesTheHook: Zy, 2026-10-14, after the pilot's
// hook refused "cat > /tmp/plan.txt" with a heredoc, a plan file for
// "mrw write PLAN" that never touched the tree. A redirection or a tee whose
// target is an absolute path landing outside the tree passes; one into the
// tree, through a link or a dangling link included, is refused, and so is any
// target the hook cannot resolve: a relative path (the Bash tool keeps its own
// working directory across calls), a quoted path, an expansion. With no tree
// known, every target is inside.
func TestAWriteOutsideTheTaskTreePassesTheHook(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bench runs on the operator's macOS or Linux machine; Windows is out of ADR-156's scope")
	}
	scratch := t.TempDir()
	tree := filepath.Join(scratch, "tree")
	if err := os.Mkdir(tree, 0o755); err != nil {
		t.Fatal(err)
	}
	plan := filepath.Join(scratch, "plan.txt")
	into := filepath.Join(scratch, "into")
	dangling := filepath.Join(scratch, "dangling")
	flink := filepath.Join(scratch, "flink")
	deep := filepath.Join(scratch, "deep")
	if err := os.Mkdir(filepath.Join(tree, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "f.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for link, to := range map[string]string{into: tree, dangling: filepath.Join(tree, "new.txt"), flink: filepath.Join(tree, "f.txt"), deep: filepath.Join(tree, "a")} {
		if err := os.Symlink(to, link); err != nil {
			t.Fatal(err)
		}
	}
	// A hard link outside the tree is another name for a file in it (the review
	// of #413); a file outside that is no such name passes.
	alias := filepath.Join(scratch, "alias")
	if err := os.Link(filepath.Join(tree, "f.txt"), alias); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(scratch, "existing.txt")
	if err := os.WriteFile(existing, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hardlink := "echo x > " + alias
	// A dot-dot after a link is resolved by the shell where the link leads:
	// deep/../f.txt is the tree's f.txt, though its spelling says otherwise.
	dotdot := deep + string(filepath.Separator) + ".." + string(filepath.Separator) + "f.txt"
	// On a filesystem that folds case, TREE is the tree.
	folded := filepath.Join(scratch, "TREE", "f.txt")
	_, foldErr := os.Stat(filepath.Join(scratch, "TREE"))
	heredoc := " <" + "<'EOF'\n@@ a.go 1 replace\nx\nEOF"
	pass := []string{
		"cat > " + plan + heredoc,
		"printf x >> " + plan,
		"printf x | tee " + plan,
		"printf x | tee -a " + plan + " " + filepath.Join(scratch, "other.txt"),
		"echo x > " + filepath.Join(tree, "..", "plan.txt"),
		"printf x >> " + existing,
		// B fixing its own plan outside the tree (ADR-160 T6)
		"sed -i 's/body=6/body=8/' " + plan,
		"sed -i '' -e 's/a/b/' -e 's/c/d/' " + plan,
		"perl -pi -e 's/a/b/' " + plan,
	}
	refuse := []string{
		"echo x > " + filepath.Join(tree, "f.txt"),
		"echo x > " + tree,
		"echo x > f.txt",
		"echo x > ../plan.txt",
		"echo x > " + filepath.Join(into, "f.txt"),
		"echo x > " + dangling,
		"echo x > \"" + plan + "\"",
		"echo x > $" + "TMPDIR/plan.txt",
		"printf x | tee " + plan + " " + filepath.Join(tree, "f.txt"),
		"printf x | tee f.txt",
		"echo x > " + flink,
		"echo x > " + dotdot,
		hardlink,
		"sed -i 's/a/b/' " + filepath.Join(tree, "f.txt"),
		"sed -i 's/a/b/' " + plan + " f.txt",
		"sed -i s/a/b/ " + plan,
		"perl -pi -e 's/a/b/' " + plan + " " + filepath.Join(tree, "f.txt"),
	}
	if foldErr == nil {
		refuse = append(refuse, "echo x > "+folded)
	}
	run := func(cmd, tree string) int {
		return Hook(bytes.NewReader(hookInput(t, "Bash", cmd)), new(bytes.Buffer), filepath.Join(t.TempDir(), "hook.log"), tree)
	}
	for _, c := range pass {
		if code := run(c, tree); code != 0 {
			t.Errorf("a write outside the tree was refused: %q (exit %d)", c, code)
		}
		if code := run(c, ""); code != 2 {
			t.Errorf("with no tree known, a write passed: %q (exit %d)", c, code)
		}
	}
	for _, c := range refuse {
		if code := run(c, tree); code != 2 {
			t.Errorf("a write into the tree, or to a target the hook cannot resolve, passed: %q (exit %d)", c, code)
		}
	}
}

// TestASlippedWriteIsFoundFromTheLedger: a file the tree shows changed that
// the attempt's own mrw ledger never recorded as written (ADR-118's `w`) is a
// write that slipped past the hook.
func TestASlippedWriteIsFoundFromTheLedger(t *testing.T) {
	needGit(t)
	tree := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tree, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "sub", "g.go"), []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"a.txt", "b.txt", "c.txt", "d.txt", "e.txt", "f.txt", "g.go", "sub/f.go"} {
		if err := os.WriteFile(filepath.Join(tree, f), []byte("one\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.name=b", "-c", "user.email=b@b", "commit", "-q", "-m", "base"}} {
		if out, err := exec.Command("git", append([]string{"-C", tree}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	for f, s := range map[string]string{"a.txt": "two\n", "b.txt": "two\n", "d.txt": "two\n", "e.txt": "three\n", "g.go": "two\n", "sub/f.go": "two\n"} {
		if err := os.WriteFile(filepath.Join(tree, f), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Remove(filepath.Join(tree, "f.txt")); err != nil {
		t.Fatal(err)
	}
	two, one := seen.SHA([]byte("two\n")), seen.SHA([]byte("one\n"))
	state := t.TempDir()
	ledger := filepath.Join(state, "mrw", "0123", "seen")
	if err := os.MkdirAll(filepath.Dir(ledger), 0o755); err != nil {
		t.Fatal(err)
	}
	// a.txt mrw wrote; d.txt mrw wrote and then read whole, which clears the
	// written mark; e.txt mrw wrote as "two" and something else then changed;
	// f.txt mrw read and something else deleted; b.txt mrw never saw.
	body := "#mrw-seen v5 mrw stub\n" + two + "  w1-1  a.txt\n" + one + "  1-1  c.txt\n" + two + "  -  d.txt\n" + two + "  w  e.txt\n" + one + "  -  f.txt\n"
	if err := os.WriteFile(ledger, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	// Each ledger's paths are relative to the root its marker names (the final
	// review of #411): sub/f.go mrw wrote under --root sub, and its read of
	// sub/g.go, whose bytes are what g.go now holds, is not a record of g.go.
	realTree, err := filepath.EvalSymlinks(tree)
	if err != nil {
		t.Fatal(err)
	}
	subLedger := filepath.Join(state, "mrw", "4567", "seen")
	if err := os.MkdirAll(filepath.Dir(subLedger), 0o755); err != nil {
		t.Fatal(err)
	}
	for p, s := range map[string]string{
		filepath.Join(filepath.Dir(ledger), "root"):    realTree + "\n",
		filepath.Join(filepath.Dir(subLedger), "root"): filepath.Join(realTree, "sub") + "\n",
		subLedger: "#mrw-seen v5 mrw stub\n" + two + "  w1-1  f.go\n" + two + "  -  g.go\n",
	} {
		if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := SlippedWrites(tree, state)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "b.txt,e.txt,f.txt,g.go" {
		t.Errorf("slipped writes: %v, want [b.txt e.txt f.txt g.go]: a.txt, d.txt and sub/f.go are what mrw last saw, c.txt is unchanged", got)
	}
	// Two ledgers holding different content for one file cannot say which mrw
	// saw last: a stale one could vouch for a later change by other means
	// (the narrow review of #411), so the check cannot run.
	if err := os.WriteFile(ledger, []byte(body+one+"  -  sub/f.go\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := SlippedWrites(tree, state); err == nil {
		t.Error("two ledgers that disagree on sub/f.go were read as if either were the last word")
	}
	if err := os.WriteFile(ledger, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	// A ledger whose root is not named cannot be read against the tree.
	if err := os.Remove(filepath.Join(filepath.Dir(subLedger), "root")); err != nil {
		t.Fatal(err)
	}
	if _, err := SlippedWrites(tree, state); err == nil {
		t.Error("a ledger with no root marker was read as if its root were the tree")
	}
}

// TestAnInterpreterIsJudgedOnItsOwnSegmentAndTheHeredocItReads: T3's pilot and
// Stage 3 saw B refused "python writes a file" for text that was not python's —
// an mrw plan holding a return annotation beside a python check, a heredoc fed
// to python that compares `a > b` (ADR-160 T6). An interpreter that writes,
// inline or through the heredoc it reads, is still refused.
func TestAnInterpreterIsJudgedOnItsOwnSegmentAndTheHeredocItReads(t *testing.T) {
	tree := t.TempDir()
	pass := []string{
		"mrw write - <<'EOF'\n@@ a.py 1 replace\ndef f(x) -> str:\n    return x\nEOF\npython3 -c \"import a; print(a.f(1))\"",
		"python3 - <<'EOF'\nfor s in ['a > b', '1 < 2']:\n    print(s, '->', s)\nEOF",
		"python3 -c \"print(1 > 0)\"",
		// a plan editing python that itself writes files, beside a python check
		"mrw write - <<'EOF'\n@@ w.py 1 replace\nopen(p, 'w').write(s)\nEOF\npython3 -c \"import w\"",
	}
	refuse := []string{
		"python3 - <<'EOF'\nopen('f','w').write('x')\nEOF",
		"python3 -c \"open('f','w').write('x')\"",
		"mrw write - <<'EOF'\n@@ a.py 1 replace\nx = 1\nEOF\npython3 -c \"open('f','w')\"",
		"perl -e 'open(F, \">out\"); print F 1'",
		// a heredoc fed to python through a pipe (the review of #434)
		"cat <<'PY' | python3\nopen('f','w').write('x')\nPY",
	}
	run := func(cmd string) int {
		return Hook(bytes.NewReader(hookInput(t, "Bash", cmd)), new(bytes.Buffer), filepath.Join(t.TempDir(), "hook.log"), tree)
	}
	for _, c := range pass {
		if code := run(c); code != 0 {
			t.Errorf("an interpreter that writes nothing was refused: %q (exit %d)", c, code)
		}
	}
	for _, c := range refuse {
		if code := run(c); code != 2 {
			t.Errorf("an interpreter that writes a file passed: %q (exit %d)", c, code)
		}
	}
}
