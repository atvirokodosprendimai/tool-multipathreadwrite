package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// ADR-145. A pre-ADR-004 ./.mrw/ is copied into the state directory by the first
// command that uses state. The migration runs from the root command's Before,
// so whatever the parser answers on its own — the version flag in every
// spelling it takes, --help — and the two commands that touch no state migrate
// nothing; a flag before the verb does not matter. The pair: a read in the same
// directory does migrate, so the run before it did not consume anything.
func TestAStartThatTouchesNoStateMigratesNothing(t *testing.T) {
	for _, argv := range [][]string{
		{"version"}, {"instructions"}, {"instructions", "--core"},
		{"-v"}, {"--v"}, {"-version"}, {"--version"},
		{"--version=true"}, {"--version=false"}, {"--version="}, {"-v=0"}, {"-v=T"},
		{"-h"}, {"--help"}, {"read", "-h"}, {"-C", ".", "version"},
	} {
		t.Run(strings.Join(argv, " "), func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("XDG_STATE_HOME", t.TempDir())
			if err := os.MkdirAll(filepath.Join(root, ".mrw"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, ".mrw", "seen"), []byte("#mrw-seen v4\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("one\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Chdir(root)
			dir, err := state.Dir(".")
			if err != nil {
				t.Fatal(err)
			}
			got := stderrOf(t, func() { runIn(t, root, argv...) })
			if strings.Contains(got, "moved") {
				t.Errorf("mrw %v migrated: %q", argv, got)
			}
			if _, err := os.Stat(filepath.Join(dir, "seen")); err == nil {
				t.Errorf("mrw %v copied the legacy seen file into %s", argv, dir)
			}
			pair := stderrOf(t, func() { runIn(t, root, "read", "f.txt") })
			if !strings.Contains(pair, "moved seen from ./.mrw/") {
				t.Errorf("the pair: a read after mrw %v did not migrate: %q", argv, pair)
			}
		})
	}
}
