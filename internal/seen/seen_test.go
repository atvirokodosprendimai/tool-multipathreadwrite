package seen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	root := t.TempDir()
	if err := Record(root, map[string]Observation{"a.go": {SHA: "aaa"}, "b.go": {SHA: "bbb"}}); err != nil {
		t.Fatal(err)
	}
	l, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if l["a.go"].SHA != "aaa" || l["b.go"].SHA != "bbb" {
		t.Errorf("ledger = %v", l)
	}
}

func TestMissingLedgerIsEmptyNotAnError(t *testing.T) {
	l, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(l) != 0 {
		t.Errorf("ledger = %v", l)
	}
}

// One command observing two files must not erase what another observed about a
// third: a read of b.go cannot invalidate an earlier read of a.go.
func TestRecordMergesRatherThanReplaces(t *testing.T) {
	root := t.TempDir()
	if err := Record(root, map[string]Observation{"a.go": {SHA: "aaa"}}); err != nil {
		t.Fatal(err)
	}
	if err := Record(root, map[string]Observation{"b.go": {SHA: "bbb"}}); err != nil {
		t.Fatal(err)
	}
	l, _ := Load(root)
	if len(l) != 2 || l["a.go"].SHA != "aaa" {
		t.Errorf("the second record dropped the first: %v", l)
	}
}

func TestRecordOverwritesTheSamePath(t *testing.T) {
	root := t.TempDir()
	_ = Record(root, map[string]Observation{"a.go": {SHA: "old"}})
	_ = Record(root, map[string]Observation{"a.go": {SHA: "new"}})
	l, _ := Load(root)
	if l["a.go"].SHA != "new" {
		t.Errorf("a.go = %+v, want the newer observation", l["a.go"])
	}
	// And exactly one line, not two.
	lp, err := ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(lp)
	if err != nil {
		t.Fatal(err)
	}
	// The header line, then exactly one record — not two records.
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 || lines[0] != header {
		t.Fatalf("ledger is not a header plus one record:\n%s", b)
	}
	if n := len(strings.Fields(lines[1])); n != 3 {
		t.Errorf("record holds %d fields, want 3 (sha, spans, path):\n%s", n, b)
	}
}

func TestSHAIsStable(t *testing.T) {
	if SHA([]byte("hello")) != SHA([]byte("hello")) {
		t.Error("SHA is not deterministic")
	}
	if SHA([]byte("hello")) == SHA([]byte("hellp")) {
		t.Error("SHA collides on a one-byte difference")
	}
}

// TestMain isolates the whole package from the user's real state directory.
// Without it these tests would write into ~/.local/state/mrw — a test suite
// that leaves state on the machine it ran on is its own small version of the
// bug ADR-004 fixes.
func TestMain(m *testing.M) {
	base, err := os.MkdirTemp("", "mrw-state-*")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_STATE_HOME", base)
	code := m.Run()
	os.RemoveAll(base)
	os.Exit(code)
}

// The reported bug, as a test: a Record must leave the working tree untouched.
func TestLedgerIsNotWrittenIntoTheRoot(t *testing.T) {
	root := t.TempDir()
	if err := Record(root, map[string]Observation{"a.go": {SHA: "aaa"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".mrw")); err == nil {
		t.Error("Record created .mrw/ in the working tree — the defect ADR-004 fixes")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("the root holds %d entry/entries after Record, want 0", len(entries))
	}
}

// A pre-ADR-004 ledger is found, and its CONTENTS ARE DISCARDED.
//
// ADR-004's migration is "one-time, additive, and never deletes", and it still
// is: nothing is removed from the tree, `.mrw/` stays where it is, and the
// working set is untouched. What changed is what the migration is TRUSTED TO
// CARRY. Up to v0.0.11 a ranged read that served nothing recorded the whole
// file, and on disk that entry is byte-identical to a legitimate whole-file
// read — so no parse-time rule can separate them, and carrying the file across
// would carry the poisoned licences with it.
//
// Discarding costs one re-read and never data: a ledger is a cache of
// observations, and its failure direction is a refusal, not a wrong write.
func TestAPreV2InTreeLedgerIsDiscardedNotTrusted(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, ".mrw")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	// TWO entries, and the count is the assertion's whole strength. Load
	// consumes line 1 as the header ALWAYS, before deciding whether it is one,
	// so a ONE-entry headerless ledger has its only record eaten either way —
	// the ledger comes back empty whether or not the discard runs, and this
	// test passed with the version check disabled. The second entry is what
	// the discard has to remove.
	body := []byte("abc123  old.go\ndef456  older.go\n")
	if err := os.WriteFile(filepath.Join(legacy, Name), body, 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := Load(root)
	if err != nil {
		t.Fatalf("a stale ledger must not be an error, or Record can never heal it: %v", err)
	}
	if len(l) != 0 {
		t.Errorf("a pre-v2 ledger was trusted: %v", l)
	}
	// Named explicitly: older.go is the entry that survives the header-line
	// consumption, so it is the one that proves the discard ran.
	if _, ok := l["older.go"]; ok {
		t.Error("the second entry survived, so nothing discarded the ledger")
	}
	stale, err := IsStale(root)
	if err != nil {
		t.Fatal(err)
	}
	if !stale {
		t.Error("IsStale did not report the pre-v2 ledger, so nothing tells the caller why")
	}

	// It must HEAL: Record calls Load, so a stale ledger returning an error
	// would stop save from ever running and the file would be stale forever.
	if err := Record(root, map[string]Observation{"new.go": {SHA: "def456"}}); err != nil {
		t.Fatal(err)
	}
	if stale, _ := IsStale(root); stale {
		t.Error("the ledger did not heal: still stale after a write")
	}
	l, _ = Load(root)
	if l["new.go"].SHA != "def456" {
		t.Errorf("the healed ledger lost the new observation: %v", l)
	}
	// And a subsequent write goes to the state dir, never back into the tree.
	b, _ := os.ReadFile(filepath.Join(legacy, Name))
	if strings.Contains(string(b), "new.go") {
		t.Error("Record wrote into the legacy in-tree ledger")
	}
}

// A legacy path may itself contain a double space, which is also the separator
// the current three-field line uses. "<sha>  my  file.go" is one path, not
// spans plus a path — and reading it as spans would silently record "nothing
// served" for a file the caller had read whole.
//
// Asserted against parseLine directly. It used to go through Load with a
// header-less file, which a v2 mrw now discards before parsing — so that route
// would have passed for the wrong reason, testing the discard rather than the
// split.
func TestALegacyPathContainingADoubleSpaceIsOnePath(t *testing.T) {
	path, obs, ok := parseLine("abc123  my  file.go")
	if !ok {
		t.Fatal("the line was not parsed at all")
	}
	if path != "my  file.go" {
		t.Errorf("path = %q, want %q", path, "my  file.go")
	}
	if !obs.Whole() {
		t.Errorf("a legacy line must read as a whole-file observation, got %s", obs.Served())
	}
}

// The header's VALUE is pinned here, deliberately with a literal rather than
// with the constant.
//
// Found by mutation: changing `header` from "#mrw-seen v2" to "#mrw-seen v3"
// SURVIVED the whole suite. Every other test writes and reads through the same
// constant, so a bump is self-consistent and invisible to them — while for a
// real user it silently discards a ledger that was never poisoned, on upgrade,
// with the notice firing for no reason.
//
// A test that used `header` here would survive the same mutation, which is the
// point: this fixture is the recorded fact that v2 is the format v0.0.12
// shipped. Bumping the constant now breaks it, and that is the intent — the
// bump is only correct when the meaning of an old file has ACTUALLY changed,
// and this test is where you say so on purpose.
func TestAV2LedgerIsAcceptedAndTheBumpIsDeliberate(t *testing.T) {
	root := t.TempDir()
	lp, err := ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(lp), 0o755); err != nil {
		t.Fatal(err)
	}
	// Literal, byte for byte, as a v0.0.12 mrw writes it. Two records, because
	// Load consumes line 1 as the header either way and a single record could
	// not tell acceptance from discard.
	body := "#mrw-seen v2\nabc123  -  kept.go\ndef456  2-4  partial.go\n"
	if err := os.WriteFile(lp, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	l, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(l) != 2 {
		t.Fatalf("a v2 ledger was not accepted: %v", l)
	}
	if o := l["kept.go"]; !o.Whole() || o.SHA != "abc123" {
		t.Errorf("whole-file record did not survive: %+v", o)
	}
	if o := l["partial.go"]; o.Whole() || !o.Covers(3, 3) || o.Covers(5, 5) {
		t.Errorf("span record did not survive intact: %+v", o)
	}
	if stale, _ := IsStale(root); stale {
		t.Error("a v2 ledger was reported stale")
	}
}

// And the round trip: what this build WRITES, this build must read back. The
// mutation above also passes a suite that never checks its own output is
// loadable — the two halves are only pinned together by asserting both.
func TestALedgerThisBuildWroteLoadsBack(t *testing.T) {
	root := t.TempDir()
	want := map[string]Observation{
		"whole.go":   {SHA: "aaa"},
		"partial.go": {SHA: "bbb", Spans: [][2]int{{2, 4}}},
		"nothing.go": {SHA: "ccc", Spans: [][2]int{}},
	}
	if err := Record(root, want); err != nil {
		t.Fatal(err)
	}
	if stale, _ := IsStale(root); stale {
		t.Fatal("this build wrote a ledger it considers stale")
	}
	got, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for path, w := range want {
		g, ok := got[path]
		if !ok {
			t.Errorf("%s did not survive the round trip", path)
			continue
		}
		if g.SHA != w.SHA || g.Whole() != w.Whole() || g.Served() != w.Served() {
			t.Errorf("%s = %+v (%s), want %+v (%s)", path, g, g.Served(), w, w.Served())
		}
	}
}

func TestLegacyInTreeLedgerIsStillRead(t *testing.T) {
	// ADR-004 moved state out of the working tree and kept READING the old
	// in-tree location. state.go still implements that fallback and its comment
	// still says "It is still READ", and between #19 and 2026-09-04 nothing
	// tested it — no test file so much as referenced LegacyDir, while ADR-004's
	// Tests table went on naming this test, which is what adr-lint reported.
	//
	// WHAT THIS DOES AND DOES NOT COVER, because the distinction is easy to
	// overstate. It covers the LOCATION fallback: with an empty state directory,
	// Load reads .mrw/seen, and Record writes the state directory instead of
	// writing back into the tree. It does NOT reconstruct a genuine
	// pre-ADR-004 ledger: state moved out of the tree in v0.0.5 and the v2
	// header arrived in v0.0.12, so a real in-tree ledger has no header and
	// today's Load discards it deliberately (seen.go, the header check). An
	// upgrade from that era therefore loses its observations and re-reads —
	// which is a decision ADR-002's stale-ledger rule already made on purpose,
	// not a gap this test hides. The fixture carries a v2 header because a
	// headerless one would test the discard, not the fallback.
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	legacy := filepath.Join(root, ".mrw")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, Name),
		[]byte(header+"\nabc123  -  old.go\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	beforeBytes, err := os.ReadFile(filepath.Join(legacy, Name))
	if err != nil {
		t.Fatal(err)
	}
	before := string(beforeBytes)

	l, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if l["old.go"].SHA != "abc123" {
		t.Fatalf("a legacy in-tree ledger was ignored: %v", l)
	}
	if !l["old.go"].Whole() {
		t.Error("the legacy entry lost its whole-file observation")
	}

	// And a later write goes to the state directory, never back into the tree:
	// ADR-004's whole point is that mrw leaves nothing behind.
	if err := Record(root, map[string]Observation{"new.go": {SHA: "def456"}}); err != nil {
		t.Fatal(err)
	}
	// Byte-for-byte, not "does it still mention old.go": the claim is that the
	// legacy file is never written or deleted, and a substring check would pass
	// on a file that had been rewritten around it.
	after, err := os.ReadFile(filepath.Join(legacy, Name))
	if err != nil {
		t.Fatalf("Record deleted the legacy ledger: %v", err)
	}
	if string(after) != before {
		t.Errorf("Record modified the legacy in-tree ledger.\n before: %q\n  after: %q", before, after)
	}
	// And the new observation really did land in the state directory.
	reloaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded["new.go"].SHA != "def456" {
		t.Errorf("the recorded observation is not readable back: %v", reloaded)
	}
}
