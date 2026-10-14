package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-143 over MCP, which the Windows peers did not try: mrw_write refuses a
// create and an unlink under .git as the CLI does, and mrw_read still serves it.
func TestAnMCPWriteIntoDotGitIsRefusedAndTheReadIsNot(t *testing.T) {
	root, _ := checkout(t, "a.txt", "one\n")
	if err := os.MkdirAll(filepath.Join(root, ".git", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "config"), []byte("[core]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, plan := range []string{
		"@@ .git/hooks/pre-commit 0 create\n#!/bin/sh\n",
		"@@ .git/config - unlink\n",
	} {
		got := structured(t, call(t, root, "mrw_write", map[string]any{"plan": plan}))
		if applied, _ := got["applied"].(bool); applied {
			t.Fatalf("mrw_write applied a write into .git: %v", got)
		}
		hunks, _ := got["hunks"].([]any)
		h, _ := hunks[0].(map[string]any)
		if h["status"] != "failed" || !strings.Contains(stringOf(h["reason"]), "inside a .git directory") {
			t.Errorf("plan %q: hunk = %v, want failed naming .git", plan, h)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-commit")); err == nil {
		t.Error("a refused mrw_write made the hook")
	}
	if b, err := os.ReadFile(filepath.Join(root, ".git", "config")); err != nil || string(b) != "[core]\n" {
		t.Errorf(".git/config = %q (%v) after refused plans", b, err)
	}
	if text := served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{".git/config:1"}})); !strings.Contains(text, "[core]") {
		t.Errorf("mrw_read of .git/config = %q, want it served", text)
	}
}
