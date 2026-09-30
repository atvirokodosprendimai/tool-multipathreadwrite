package state

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// ADR-105 T3. The two files that carry licences — the ledger and the MCP
// acknowledgement store — are synced before the rename that publishes them,
// so a power loss cannot leave the name pointing at data that never reached
// the disk. WriteSynced syncs and only then renames; Write does not sync,
// because the other state files are measurement and convenience and a sync
// costs about 4 ms (the record's Decision 3).
func TestTheLicenceFilesAreSyncedBeforeTheyAreRenamed(t *testing.T) {
	var events []string
	realSync, realRename := syncFn, renameFn
	t.Cleanup(func() { syncFn, renameFn = realSync, realRename })
	syncFn = func(f *os.File) error { events = append(events, "sync"); return realSync(f) }
	renameFn = func(a, b string) error { events = append(events, "rename"); return realRename(a, b) }

	name := filepath.Join(t.TempDir(), "seen")
	if err := WriteSynced(name, []byte("ledger\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(events, []string{"sync", "rename"}) {
		t.Errorf("WriteSynced did %q, want a sync and then the rename", events)
	}
	events = nil
	if err := Write(name, []byte("tally\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(events, []string{"rename"}) {
		t.Errorf("Write did %q, want the rename alone", events)
	}

	for _, f := range []string{"../seen/seen.go", "../mcp/ack.go"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "state.WriteSynced(") || strings.Contains(string(b), "state.Write(") {
			t.Errorf("%s does not write through state.WriteSynced alone", f)
		}
	}
	// A legacy ledger migrated into the state directory is a licence file too
	// (the review of the record): Migrate writes through WriteSynced.
	if b, err := os.ReadFile("state.go"); err != nil || !strings.Contains(string(b), "WriteSynced(to, b, 0o600)") {
		t.Errorf("Migrate does not write through WriteSynced (%v)", err)
	}
}
