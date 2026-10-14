package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-144. A ledger with a line mrw cannot parse is told on stderr by the
// commands that read it, once: the read rewrites the ledger without the line,
// so the next run has nothing to say. version prints nothing.
func TestADamagedLedgerIsToldOnceByTheCLI(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runIn(t, root, "read", "f.txt")
	p, err := seen.ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	damage := func() {
		t.Helper()
		f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteString("garbage\n"); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	damage()
	if got := stderrOf(t, func() { runIn(t, root, "version") }); strings.Contains(got, "read ledger") {
		t.Errorf("version printed the ledger notice, though it never reads the ledger: %q", got)
	}
	got := stderrOf(t, func() { runIn(t, root, "read", "f.txt") })
	if !strings.Contains(got, "1 line(s) of the read ledger could not be understood") || strings.Count(strings.TrimSpace(got), "\n") != 0 {
		t.Errorf("read printed %q, want one line counting the bad line", got)
	}
	if again := stderrOf(t, func() { runIn(t, root, "read", "f.txt") }); strings.Contains(again, "read ledger") {
		t.Errorf("the next read told it again: %q", again)
	}
}

// ADR-144 T2 (the Codex review of #390). Only the commands that read the ledger
// tell its damage: check, iter and mcp neither consume nor save it, so the
// sentence would repeat for ever there (and mcp resolves its root after Before
// has run). seen reads it and is where one goes to look, so it tells.
func TestOnlyTheCommandsThatReadTheLedgerTellItsDamage(t *testing.T) {
	for verb, want := range map[string]bool{"read": true, "write": true, "seen": true, "check": false, "iter": false, "mcp": false, "version": false} {
		if got := tellsLedgerDamage(verb); got != want {
			t.Errorf("tellsLedgerDamage(%q) = %v, want %v", verb, got, want)
		}
	}
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runIn(t, root, "read", "f.txt")
	p, err := seen.ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("garbage\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if got := stderrOf(t, func() { runIn(t, root, "iter") }); strings.Contains(got, "read ledger") {
		t.Errorf("iter told the ledger's damage, though it neither reads nor saves it: %q", got)
	}
	if got := stderrOf(t, func() { runIn(t, root, "seen") }); !strings.Contains(got, "1 line(s) of the read ledger could not be understood") {
		t.Errorf("seen did not tell the damage: %q", got)
	}
}
