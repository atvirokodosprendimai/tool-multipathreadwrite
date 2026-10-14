package bench

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
		code := Hook(bytes.NewReader(hookInput(t, "Bash", cmd)), &stderr, log)
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
	if code := Hook(bytes.NewReader(hookInput(t, "Glob", "")), new(bytes.Buffer), filepath.Join(t.TempDir(), "l")); code != 0 {
		t.Errorf("a tool other than Bash was refused: exit %d", code)
	}
}

// TestASlippedWriteIsFoundFromTheLedger: a file the tree shows changed that
// the attempt's own mrw ledger never recorded as written (ADR-118's `w`) is a
// write that slipped past the hook.
func TestASlippedWriteIsFoundFromTheLedger(t *testing.T) {
	needGit(t)
	tree := t.TempDir()
	for _, f := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(tree, f), []byte("one\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.name=b", "-c", "user.email=b@b", "commit", "-q", "-m", "base"}} {
		if out, err := exec.Command("git", append([]string{"-C", tree}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	for _, f := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(tree, f), []byte("two\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	state := t.TempDir()
	ledger := filepath.Join(state, "mrw", "0123", "seen")
	if err := os.MkdirAll(filepath.Dir(ledger), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ledger, []byte("#mrw-seen v5 mrw stub\nabc  w1-1  a.txt\ndef  1-1  c.txt\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := SlippedWrites(tree, state)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "b.txt" {
		t.Errorf("slipped writes: %v, want [b.txt] (a.txt was written by mrw; c.txt was only read and is unchanged)", got)
	}
}
