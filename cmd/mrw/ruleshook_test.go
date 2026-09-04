package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestThePathScopedRulesHookDeliversOnAnMrwRead is ADR-022's Enforced-by. It
// drives .claude/hooks/rules-on-read.py the way Claude Code does — JSON on
// stdin, the documented envelope on stdout — with the session's cwd in a
// SUBDIRECTORY of the project, which is where the first cut delivered nothing
// (it took cwd for the project root). A second call in the same session must
// deliver nothing: once per rule per session is the harness's own behaviour.
//
// It skips where python3 is absent, which is the Windows runner; the contract
// row on Linux never skips.
func TestThePathScopedRulesHookDeliversOnAnMrwRead(t *testing.T) {
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is not on PATH; the hook cannot run here")
	}
	hook, err := filepath.Abs(filepath.Join("..", "..", ".claude", "hooks", "rules-on-read.py"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(hook); err != nil {
		t.Fatalf("the hook this record is about is not where settings.json points: %v", err)
	}
	proj, home := t.TempDir(), t.TempDir()
	for name, body := range map[string]string{
		".claude/rules/scoped.md": "---\npaths:\n  - \"docs/adr/**\"\n---\n\nSCOPED RULE BODY\n",
		"docs/adr/x.md":           "a record\n",
		"pkg/keep.txt":            "",
	} {
		full := filepath.Join(proj, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run := func(session, command string) string {
		in, _ := json.Marshal(map[string]any{
			"hook_event_name": "PostToolUse", "session_id": session,
			"cwd": filepath.Join(proj, "pkg"), "tool_name": "Bash",
			"tool_input": map[string]any{"command": command},
		})
		c := exec.Command(py, hook)
		c.Stdin = strings.NewReader(string(in))
		c.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+proj, "HOME="+home, "XDG_CACHE_HOME="+filepath.Join(home, ".cache"))
		out, err := c.Output()
		if err != nil {
			t.Fatalf("the hook exited non-zero, which would fail the turn: %v", err)
		}
		return string(out)
	}
	first := run("s1", "mrw read ../docs/adr/x.md:1")
	var env struct {
		H struct {
			Name string `json:"hookEventName"`
			Ctx  string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(first), &env); err != nil {
		t.Fatalf("from a subdirectory, no documented envelope came back for a read of ../docs/adr/x.md: %q", first)
	}
	if env.H.Name != "PostToolUse" || !strings.Contains(env.H.Ctx, "SCOPED RULE BODY") {
		t.Fatalf("the rule for docs/adr/** was not delivered: %q", first)
	}
	if again := run("s1", "mrw read ../docs/adr/x.md:1"); again != "" {
		t.Fatalf("the same rule was delivered twice in one session: %q", again)
	}
}
