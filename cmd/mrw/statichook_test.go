package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestTheStaticHookReportsAfterACommit is ADR-088's Enforced-by. It drives
// .claude/hooks/static-after-commit.py the way Claude Code does — JSON on
// stdin, the documented envelope on stdout — against a scratch repository whose
// scripts/static.sh is a stand-in, so what is asserted is the trigger and the
// envelope, not the analysers: a command that ran `git commit`, after a commit
// landed, hands the script's verdict and output to the model, failing or clean,
// and a closed stdout still exits 0.
func TestTheStaticHookReportsAfterACommit(t *testing.T) {
	failing := newStaticHookRepo(t, "echo FINDING-088; exit 1")
	failing.commit("second")
	// A heredoc whose double-quoted delimiter escapes a $ ends at E$OF, and the
	// commit after it is a command.
	if ctx := failing.context("cat <<\"E\\$OF\"\ntext\nE$OF\ngit commit -m second"); !strings.Contains(ctx, "FINDING-088") || !strings.Contains(ctx, "FAILED") {
		t.Fatalf("a commit whose analysis failed did not hand the finding over: %q", ctx)
	}

	clean := newStaticHookRepo(t, "echo nothing found; exit 0")
	clean.commit("second")
	// A quoted & does not end the command: this is still `git commit`.
	if ctx := clean.context("git -c user.name='Tom & Sam' commit -m second"); !strings.Contains(ctx, "clean") || strings.Contains(ctx, "FAILED") {
		t.Fatalf("a commit whose analysis passed did not say so: %q", ctx)
	}

	closed := newStaticHookRepo(t, "echo FINDING-088; exit 1")
	closed.commit("second")
	if err := closed.withClosedStdout("git commit -m second"); err != nil {
		t.Fatalf("with its stdout closed the hook did not exit 0: %v", err)
	}
}

// TestTheStaticHookIsQuietWhenHEADDidNotMove pins what the hook does NOT run
// for: a command that is not a commit, however much it mentions one; a commit
// already analysed; a HEAD whose newest move was not a commit. And a command in
// another session cannot consume a commit that is still to be analysed.
func TestTheStaticHookIsQuietWhenHEADDidNotMove(t *testing.T) {
	h := newStaticHookRepo(t, "echo FINDING-088; exit 1")
	h.commit("moved")
	for _, cmd := range []string{"git status", "git log --grep commit", `echo "git commit"`, `echo ";" git commit`, "cat <<123\ngit commit\n123", "cat <<\"E'OF\"\nEOF\ngit commit\nE'OF"} {
		if out := h.raw(cmd); out != "" {
			t.Fatalf("%q is not a commit and ran the analysis: %q", cmd, out)
		}
	}
	// A backslash-newline joins the lines: this is still `git commit`.
	if ctx := h.context("git \\\ncommit -m moved"); !strings.Contains(ctx, "FINDING-088") {
		t.Fatalf("the commit was consumed by the commands before it: %q", ctx)
	}
	if out := h.raw("git commit -m nothing-new"); out != "" {
		t.Fatalf("a commit already analysed ran the analysis again: %q", out)
	}
	h.git("checkout", "-q", "-b", "other")
	if out := h.raw("git commit -m on-other"); out != "" {
		t.Fatalf("a HEAD whose newest move was a checkout ran the analysis: %q", out)
	}
}

type staticHookRepo struct {
	t              *testing.T
	py, hook, root string
	env            []string
}

// newStaticHookRepo builds a scratch repository with one commit and a
// scripts/static.sh running body, and resolves the hook from
// .claude/settings.json so an unregistered hook fails here too. It skips where
// python3, git or a POSIX script cannot run — the Windows runner.
func newStaticHookRepo(t *testing.T, body string) *staticHookRepo {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in scripts/static.sh is a POSIX script")
	}
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is not on PATH; the hook cannot run here")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH")
	}
	h := &staticHookRepo{t: t, py: py, hook: staticHookFromSettings(t), root: t.TempDir()}
	h.env = append(os.Environ(),
		"CLAUDE_PROJECT_DIR="+h.root, "TMPDIR="+t.TempDir(),
		"GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
	script := filepath.Join(h.root, "scripts", "static.sh")
	if err := os.MkdirAll(filepath.Dir(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	h.git("init", "-q")
	h.commit("first")
	return h
}

func (h *staticHookRepo) git(args ...string) {
	h.t.Helper()
	c := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "core.hooksPath=" + os.DevNull, "-c", "commit.gpgsign=false"}, args...)...)
	c.Dir, c.Env = h.root, h.env
	if out, err := c.CombinedOutput(); err != nil {
		h.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func (h *staticHookRepo) commit(msg string) {
	h.t.Helper()
	if err := os.WriteFile(filepath.Join(h.root, "f.txt"), []byte(msg+"\n"), 0o644); err != nil {
		h.t.Fatal(err)
	}
	h.git("add", "-A")
	h.git("commit", "-q", "-m", msg)
}

// raw runs the hook for one Bash call and returns what it printed.
func (h *staticHookRepo) raw(command string) string {
	h.t.Helper()
	in, _ := json.Marshal(map[string]any{
		"hook_event_name": "PostToolUse", "session_id": "s", "cwd": h.root,
		"tool_name": "Bash", "tool_input": map[string]any{"command": command},
	})
	c := exec.Command(h.py, h.hook)
	c.Stdin, c.Env = strings.NewReader(string(in)), h.env
	out, err := c.Output()
	if err != nil {
		h.t.Fatalf("the hook exited non-zero, which would fail the turn: %v", err)
	}
	return string(out)
}

// withClosedStdout runs the hook with a stdout nobody reads and returns how it
// exited.
func (h *staticHookRepo) withClosedStdout(command string) error {
	h.t.Helper()
	in, _ := json.Marshal(map[string]any{
		"hook_event_name": "PostToolUse", "session_id": "s", "cwd": h.root,
		"tool_name": "Bash", "tool_input": map[string]any{"command": command},
	})
	r, w, err := os.Pipe()
	if err != nil {
		h.t.Fatal(err)
	}
	_ = r.Close()
	defer func() { _ = w.Close() }()
	c := exec.Command(h.py, h.hook)
	c.Stdin, c.Stdout, c.Env = strings.NewReader(string(in)), w, h.env
	return c.Run()
}

// context runs the hook and returns the additionalContext of its envelope,
// "" when it printed nothing.
func (h *staticHookRepo) context(command string) string {
	h.t.Helper()
	out := h.raw(command)
	if out == "" {
		return ""
	}
	var env struct {
		H struct {
			Name string `json:"hookEventName"`
			Ctx  string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil || env.H.Name != "PostToolUse" {
		h.t.Fatalf("the hook printed something that is not the documented envelope: %q", out)
	}
	return env.H.Ctx
}

// staticHookFromSettings returns the file .claude/settings.json runs as the
// post-commit analysis hook, resolved the way Claude Code resolves it.
func staticHookFromSettings(t *testing.T) string {
	t.Helper()
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(repo, ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var s struct {
		Hooks map[string][]struct {
			Matcher string `json:"matcher"`
			Hooks   []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	for _, e := range s.Hooks["PostToolUse"] {
		for _, h := range e.Hooks {
			if strings.Contains(h.Command, "static-after-commit.py") && strings.Contains(e.Matcher, "Bash") {
				return filepath.Join(repo, ".claude", "hooks", "static-after-commit.py")
			}
		}
	}
	t.Fatal(".claude/settings.json registers no static-after-commit.py hook on Bash")
	return ""
}
