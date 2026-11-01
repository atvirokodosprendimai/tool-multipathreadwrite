package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The concurrent --dir row of contract §293, failed once on main 7d177d24: a
// racing run's create probe (probeName, ADR-086) is a file for a moment, so this
// run's Lstat finds SKILL.md and its open then answers "no such file". That is a
// race lost, not a failure: --dir judges again and installs.
func TestSkillDirTakesAVanishedProbeForARace(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	d := t.TempDir()
	target := filepath.Join(d, "mrw", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	defer func(was func(string) ([]byte, error)) { readSkill = was }(readSkill)
	readSkill = func(p string) ([]byte, error) {
		calls++
		if calls == 1 {
			_ = os.Remove(p) // the probe's owner removes it between Lstat and open
		}
		return readSkillFile(p)
	}
	var out bytes.Buffer
	if err := installSkill(d, &out); err != nil {
		t.Fatalf("installSkill after a vanished probe: %v, want the race judged again", err)
	}
	if !strings.HasPrefix(out.String(), "created ") || calls != 1 {
		t.Errorf("out %q after %d read(s), want created after one", out.String(), calls)
	}
}
