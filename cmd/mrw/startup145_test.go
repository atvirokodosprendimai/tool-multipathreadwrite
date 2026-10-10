package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The wiring, not only the predicate: a legacy ./.mrw/seen is copied into the
// state directory by any command but the four, and by none of the four.
func TestTheLegacyMigrationRunsForOtherCommandsOnly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if err := os.MkdirAll(filepath.Join(root, ".mrw"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".mrw", "seen"), []byte("#mrw-seen v4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	if moved, err := migrateLegacyState([]string{"version"}); err != nil || len(moved) != 0 {
		t.Errorf("version migrated %q, %v; want nothing", moved, err)
	}
	if moved, err := migrateLegacyState([]string{"read", "a.go"}); err != nil || len(moved) != 1 || moved[0] != "seen" {
		t.Errorf("read migrated %q, %v; want the legacy seen file", moved, err)
	}
}

// ADR-145. main migrates a legacy ./.mrw/ before it parses anything, so the
// first argument decides whether the start touches state. The install check and
// the instructions print do not; stats reads what the migration moves, and a
// flag before the verb is not seen before the parse.
func TestVersionAndInstructionsMigrateNothing(t *testing.T) {
	for _, c := range []struct {
		args []string
		want bool
	}{
		{[]string{"version"}, true},
		{[]string{"-v"}, true},
		{[]string{"--version"}, true},
		{[]string{"instructions"}, true},
		{[]string{"instructions", "--core"}, true},
		{[]string{"read", "a.go"}, false},
		{[]string{"write", "-"}, false},
		{[]string{"stats"}, false},
		{[]string{"mcp"}, false},
		{[]string{"-C", "dir", "version"}, false},
		{[]string{}, false},
	} {
		if got := startsWithoutState(c.args); got != c.want {
			t.Errorf("startsWithoutState(%q) = %v, want %v", c.args, got, c.want)
		}
	}
}
