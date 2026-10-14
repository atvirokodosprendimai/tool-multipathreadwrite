package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// stderrOf runs fn with os.Stderr on a pipe and returns what it printed there.
func stderrOf(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	fn()
	os.Stderr = old
	_ = w.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// ADR-136. The stale-ledger notice was 90 words of version history, printed for
// every subcommand, `version` and `stats` included, though neither reads the
// ledger. It is one sentence naming both causes and the remedy, and only the
// commands that use the ledger print it.
func TestTheStaleNoticeIsOneSentenceAndSilentOnVersionStatsInstructions(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	staleLedger := func() {
		t.Helper()
		sha := strings.Repeat("0", 64)
		if err := seen.Record(root, map[string]seen.Observation{"f.txt": {SHA: sha}}); err != nil {
			t.Fatal(err)
		}
		p, err := seen.ReadPath(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("#not-a-ledger\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, argv := range [][]string{{"version"}, {"instructions"}, {"stats"}} {
		staleLedger()
		if got := stderrOf(t, func() { runIn(t, root, argv...) }); strings.Contains(got, "read ledger") {
			t.Errorf("mrw %s printed the stale-ledger notice, though it never reads the ledger:\n%s", argv[0], got)
		}
	}
	staleLedger()
	got := stderrOf(t, func() { runIn(t, root, "read", "f.txt") })
	if n := strings.Count(strings.TrimSpace(got), "\n"); n != 0 || !strings.Contains(got, "written by an older mrw") || !strings.Contains(got, "line endings") {
		t.Errorf("read printed %q, want one line naming both causes", got)
	}
	if len(strings.Fields(got)) > 40 {
		t.Errorf("the notice is %d words, want a sentence of at most 40: %s", len(strings.Fields(got)), got)
	}
	staleLedger()
	if got := stderrOf(t, func() { runIn(t, root, "write", "--no-check", "--dry-run", filepath.Join(root, "none.plan")) }); !strings.Contains(got, "written by an older mrw") {
		t.Errorf("write did not announce the stale ledger: %q", got)
	}
}
