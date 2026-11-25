package main

import (
	"os"
	"testing"
)

// TestMain isolates the whole package from the user's real state directory.
// Every in-process run passes the root command's Before hook, which asks
// seen.IsStale and so creates the root's state directory even for a read; a
// test tree left there is a dead entry in ~/.local/state/mrw (BACKLOG, ADR-034).
// The per-test helpers that pin XDG_STATE_HOME only when it is unset keep
// working: every root here is its own t.TempDir, and state is keyed by root.
func TestMain(m *testing.M) {
	base, err := os.MkdirTemp("", "mrw-state-*")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_STATE_HOME", base)
	// ADR-218: a developer's own MRW_SHOWN=off, which a host sets for a session,
	// would make every read here licence-only. The tests that mean it set it.
	os.Unsetenv("MRW_SHOWN")
	code := m.Run()
	os.RemoveAll(base)
	os.Exit(code)
}
