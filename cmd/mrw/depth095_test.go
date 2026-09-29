package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
)

// depthTree holds a Go file and a markdown file, both read, and a harness
// whose check appends the depth it sees to checked.
func depthTree(t *testing.T) string {
	t.Helper()
	root := grepTree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\n",
		"notes.md":              "# notes\nline two\n",
		".quality-harness.json": "{\"check\":\"echo $MRW_STEP_DEPTH >> checked\",\"steps\":{\"a\":\"echo a >> log\"}}",
	})
	if _, err := readIn(t, root, "a.go", "notes.md"); err != nil {
		t.Fatal(err)
	}
	return root
}

// snapshot is every regular file under root and its bytes.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	got := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		got[p] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func exists(root, name string) bool {
	_, err := os.Stat(filepath.Join(root, name))
	return err == nil
}

// ADR-095 T1. At MRW_STEP_DEPTH 8 a write whose check would be due is refused,
// exit 2, before anything is written or run, judged over the plan's own paths:
// a .go replace, a .go unlink, a .go renamed to .md (by its source), a .md
// renamed to .go (by its destination alone), --check on prose, and a code write
// under the check a go.mod infers. A write that starts no check lands as on
// v1.31.0; at 7 the write lands and its check runs at 8.
func TestAWriteWhoseCheckIsDueIsRefusedAtTheDepthLimit(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("MRW_STEP_DEPTH", "8")
	for _, c := range []struct {
		name string
		args []string
		plan string
	}{
		{"a .go replace", nil, goPlan},
		{"a .go unlink", nil, "@@ a.go - unlink\n"},
		{"a .go renamed to .md", nil, "@@ a.go - rename\nb.md\n"},
		{"a .md renamed to .go", nil, "@@ notes.md - rename\nnotes.go\n"},
		{"--check on prose", []string{"--check"}, mdPlan},
	} {
		root := depthTree(t)
		before := snapshot(t, root)
		out, code := runIn(t, root, append(append([]string{"write"}, c.args...), planFile(t, c.plan))...)
		if code != exitUsage || !strings.Contains(out, "MRW_STEP_DEPTH=8") || !strings.Contains(out, "--no-check") {
			t.Errorf("%s: exit %d, want 2 naming MRW_STEP_DEPTH=8 and --no-check:\n%s", c.name, code, out)
		}
		if after := snapshot(t, root); len(after) != len(before) || exists(root, "checked") {
			t.Errorf("%s: the tree changed or the check ran:\n%s", c.name, out)
		} else {
			for p, b := range before {
				if after[p] != b {
					t.Errorf("%s: %s changed", c.name, p)
				}
			}
		}
	}

	t.Run("the inferred check", func(t *testing.T) {
		root := grepTree(t, map[string]string{"go.mod": "module a\n\ngo 1.26\n", "a.go": "package a\nfunc A() {}\n"})
		plan := primed(t, root)
		out, code := runIn(t, root, "write", plan)
		if b, _ := os.ReadFile(filepath.Join(root, "a.go")); code != exitUsage || strings.Contains(string(b), "_ = 1") || !strings.Contains(out, "MRW_STEP_DEPTH=8") {
			t.Errorf("exit %d, a.go %q, want 2 and unchanged:\n%s", code, b, out)
		}
	})

	t.Run("--json and the tally", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := depthTree(t)
		out, code := writeIn(t, root, "--json", planFile(t, goPlan))
		jsonRefusal(t, out, code, "MRW_STEP_DEPTH=8")
		var doc map[string]any
		if err := json.Unmarshal([]byte(out), &doc); err == nil {
			if _, ok := doc["then"]; ok {
				t.Errorf("the refusal carries then:\n%s", out)
			}
		}
		tally, err := authoring.Load(root)
		if err != nil || tally["refused_apply"] != 1 || tally["applied"] != 0 {
			t.Errorf("tally %v %v, want one refused_apply and no applied", tally, err)
		}
	})

	t.Run("a write that starts no check lands", func(t *testing.T) {
		for _, c := range []struct {
			name string
			args []string
			plan string
		}{
			{"--no-check", []string{"--no-check"}, goPlan},
			{"--dry-run", []string{"--dry-run"}, goPlan},
			{"a .md-only plan", nil, mdPlan},
		} {
			root := depthTree(t)
			out, code := runIn(t, root, append(append([]string{"write"}, c.args...), planFile(t, c.plan))...)
			if code != 0 || exists(root, "checked") {
				t.Errorf("%s: exit %d, checked %v, want 0 and no check:\n%s", c.name, code, exists(root, "checked"), out)
			}
		}
		root := grepTree(t, map[string]string{"a.go": "package a\nfunc A() {}\n"})
		if out, code := runIn(t, root, "write", primed(t, root)); code != 0 {
			t.Errorf("a .go write in a tree with no check: exit %d:\n%s", code, out)
		}
	})

	t.Run("at 7 the write lands and its check runs at 8", func(t *testing.T) {
		t.Setenv("MRW_STEP_DEPTH", "7")
		root := depthTree(t)
		out, code := runIn(t, root, "write", planFile(t, goPlan))
		b, _ := os.ReadFile(filepath.Join(root, "checked"))
		if code != 0 || strings.TrimSpace(string(b)) != "8" {
			t.Errorf("exit %d, the check saw %q, want 0 and 8:\n%s", code, b, out)
		}
	})
}

// ADR-095 T1. At MRW_STEP_DEPTH 8 mrw check is refused in every form, before
// its check runs; under --json the refusal is one document holding only its
// error. At 7, and under a value mrw did not write, it runs.
func TestACheckIsRefusedAtTheDepthLimit(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := depthTree(t)
	if out, code := runIn(t, root, "iter", "add", "a.go"); code != 0 {
		t.Fatalf("iter add: exit %d:\n%s", code, out)
	}
	t.Setenv("MRW_STEP_DEPTH", "8")
	for _, args := range [][]string{{"--full"}, {"a.go"}, {}, {"--then", "a", "--full"}} {
		out, code := runIn(t, root, append([]string{"check"}, args...)...)
		if code != exitUsage || !strings.Contains(out, "MRW_STEP_DEPTH") || exists(root, "checked") || exists(root, "log") {
			t.Errorf("check %v at 8: exit %d, checked %v, want 2 naming MRW_STEP_DEPTH and nothing run:\n%s", args, code, exists(root, "checked"), out)
		}
	}
	out, code := runIn(t, root, "check", "--full", "--json")
	var doc map[string]any
	if err := json.NewDecoder(strings.NewReader(out)).Decode(&doc); err != nil || code != exitUsage {
		t.Fatalf("check --json at 8: exit %d, %v:\n%s", code, err, out)
	}
	if e, _ := doc["error"].(string); len(doc) != 1 || !strings.Contains(e, "MRW_STEP_DEPTH") {
		t.Errorf("check --json at 8 is not one document holding only its error: %v", doc)
	}
	for _, v := range []string{"7", "x"} {
		t.Setenv("MRW_STEP_DEPTH", v)
		if out, code := runIn(t, root, "check", "--full"); code != 0 {
			t.Errorf("check --full at %q: exit %d:\n%s", v, code, out)
		}
	}
}

// ADR-095 T1. A check that runs mrw check again stops at the limit: started
// with no depth, the chain records levels 1 to 8 and no ninth, and exits 3.
// The script stops itself past 12 levels, so a missing guard fails here
// instead of recursing without end.
func TestACheckThatRunsMrwAgainStopsAtTheDepthLimit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the check is a POSIX shell script that runs the built binary; contract §181 cannot run there either")
	}
	needShell(t)
	bin := filepath.Join(t.TempDir(), "mrw")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	root := grepTree(t, map[string]string{
		".quality-harness.json": "{\"check\":\"sh rec.sh\"}",
		"rec.sh": "echo \"$MRW_STEP_DEPTH\" >> depth\n" +
			"[ \"$(wc -l < depth)\" -gt 12 ] && exit 97\n" +
			"exec '" + bin + "' check --full\n",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, bin, "-C", root, "check", "--full")
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "MRW_STEP_DEPTH=") {
			c.Env = append(c.Env, kv)
		}
	}
	c.Env = append(c.Env, "XDG_STATE_HOME="+t.TempDir())
	out, err := c.CombinedOutput()
	code := -1
	if c.ProcessState != nil {
		code = c.ProcessState.ExitCode()
	}
	b, _ := os.ReadFile(filepath.Join(root, "depth"))
	if got := strings.Fields(string(b)); code != exitCheckFailed || strings.Join(got, " ") != "1 2 3 4 5 6 7 8" {
		t.Errorf("exit %d (%v), levels %q, want 3 and 1 to 8:\n%s", code, err, b, out)
	}
}
