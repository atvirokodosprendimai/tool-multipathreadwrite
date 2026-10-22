package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-165 T1. `mrw read --root DIR f` was refused as "flag provided but not
// defined: -root", true and silent about the fix: --root is mrw's own flag and
// goes before the subcommand. A Codex agent lost a turn to it (2026-10-21).
// urfave makes a root flag persistent, so most subcommands take --root after
// the verb; read does not, because its own -C (context) clashes with the root
// flag's alias and urfave then skips the root flag whole (flagKinds). So the
// test asks every subcommand: either the root flag is taken there, or its
// refusal ends with the home spelling — and read must be the second kind.
func TestAGlobalFlagTypedAfterTheSubcommandIsPointedHome(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := rootCommand()
	var names []string
	for _, f := range root.Flags {
		if !applicationFlag(f) {
			continue
		}
		names = append(names, f.Names()[0])
	}
	if len(names) == 0 {
		t.Fatal("the root command defines no flag of its own; this test would pass by trying none")
	}
	pointed := 0
	for _, name := range names {
		for _, sub := range root.Commands {
			want := "; --" + name + " goes before the subcommand: mrw --" + name + " DIR " + sub.Name + " …"
			stdout, err := runSplit(t, sub.Name, "--"+name, ".", "--help")
			if err == nil || !strings.Contains(err.Error(), "not defined: --"+name+" ") {
				continue // taken after the verb: nothing to point at
			}
			pointed++
			if exitCode(err) != exitUsage || !strings.HasSuffix(err.Error(), want) || stdout != "" {
				t.Errorf("%s --%s: err %v (exit %d), stdout %q; want exit %d ending %q", sub.Name, name, err, exitCode(err), stdout, exitUsage, want)
			}
		}
	}
	if _, err := runSplit(t, "read", "--root", ".", "x"); err == nil || !strings.Contains(err.Error(), "goes before the subcommand: mrw --root DIR read …") {
		t.Errorf("read --root, the case the report met: %v", err)
	}
	if pointed == 0 {
		t.Fatal("no subcommand refused a root flag; this test would pass by trying none")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if stdout, err := runSplit(t, "--root", dir, "read", "f.txt:1"); err != nil || !strings.Contains(stdout, "1| one") {
		t.Errorf("the home spelling stopped working: err %v, stdout %q", err, stdout)
	}
	// -C is read's own context flag, so it is never refused there.
	if _, err := runSplit(t, "--root", dir, "read", "-C", "1", "f.txt:1"); err != nil && strings.Contains(err.Error(), "goes before the subcommand") {
		t.Errorf("read's own -C was pointed home: %v", err)
	}
	// ADR-164's -v pointer is not a root flag's and keeps its sentence.
	if _, err := runSplit(t, "read", "-v", "f.txt"); err == nil || !strings.Contains(err.Error(), "there is no inverted match") || strings.Contains(err.Error(), "goes before") {
		t.Errorf("read -v: %v", err)
	}
	// ADR-097's subcommand-named flag keeps precedence.
	if _, err := runSplit(t, "--instructions"); err == nil || !strings.Contains(err.Error(), "the subcommand is `mrw instructions`") {
		t.Errorf("--instructions: %v", err)
	}
	// A flag nobody defines gets neither pointer.
	if _, err := runSplit(t, "read", "--zz", "f.txt"); err == nil || strings.Contains(err.Error(), "goes before") || strings.Contains(err.Error(), "; ") {
		t.Errorf("read --zz: %v", err)
	}
}
