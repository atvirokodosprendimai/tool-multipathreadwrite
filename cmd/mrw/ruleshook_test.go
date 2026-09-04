package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
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
	hook := hookFromSettings(t)
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

// hookFromSettings returns the hook file .claude/settings.json wires for
// exactly the four tools, resolved the way Claude Code resolves it —
// ${CLAUDE_PROJECT_DIR} is this checkout. The command shape it accepts is the
// one the file has, `python3 "${CLAUDE_PROJECT_DIR}/<path>"`; any other shape
// is a wiring change this record has to look at, and fails here rather than
// silently in a session. A test that stat'd a path of its own would pass
// with the settings entry pointing at nothing.
func hookFromSettings(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("no settings.json wires the hook: %v", err)
	}
	var s struct {
		Hooks struct {
			PostToolUse []struct {
				Matcher string `json:"matcher"`
				Hooks   []struct {
					Command string `json:"command"`
				} `json:"hooks"`
			} `json:"PostToolUse"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"Bash": true, "Write": true, "mcp__mrw__mrw_read": true, "mcp__mrw__mrw_write": true}
	shape := regexp.MustCompile(`^python3 "\$\{CLAUDE_PROJECT_DIR\}/([^"]+)"$`)
	for _, e := range s.Hooks.PostToolUse {
		got := map[string]bool{}
		for _, m := range strings.Split(e.Matcher, "|") {
			got[m] = true
		}
		if !reflect.DeepEqual(got, want) || len(e.Hooks) != 1 {
			continue
		}
		m := shape.FindStringSubmatch(e.Hooks[0].Command)
		if m == nil {
			t.Fatalf("the hook command is not a shape this test can resolve: %q", e.Hooks[0].Command)
		}
		p := filepath.Join(root, filepath.FromSlash(m[1]))
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("settings.json wires a hook file that is not there: %v", err)
		}
		return p
	}
	t.Fatal("no PostToolUse entry matches exactly Bash|Write|mcp__mrw__mrw_read|mcp__mrw__mrw_write")
	return ""
}
