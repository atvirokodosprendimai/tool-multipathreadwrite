package authoring

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// tallyPath is where the tally lands for a root, via the same resolver the
// package uses — reading it from the state package rather than rebuilding the
// path keeps the test from asserting a layout ADR-004 owns.
func tallyPath(t *testing.T, root string) string {
	t.Helper()
	dirLine := os.Getenv("XDG_STATE_HOME")
	if dirLine == "" {
		t.Fatal("XDG_STATE_HOME must be pinned by the test, or this writes to the developer's real state")
	}
	matches, err := filepath.Glob(filepath.Join(dirLine, "mrw", "*", "authoring"))
	if err != nil || len(matches) == 0 {
		return ""
	}
	return matches[0]
}

func newRoot(t *testing.T) string {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	return t.TempDir()
}

func TestTallyRoundTripsThroughLoad(t *testing.T) {
	root := newRoot(t)
	if err := Record(root, Applied); err != nil {
		t.Fatal(err)
	}
	got, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got["applied"] != 1 {
		t.Errorf("applied = %d, want 1 — what was recorded did not come back", got["applied"])
	}
	if got.Plans() != 1 {
		t.Errorf("Plans() = %d, want 1", got.Plans())
	}
}

// The vocabulary must not collapse: six outcomes are six counters, and a
// category is recorded only for the outcome it refines.
func TestTallyCountsEachOutcomeSeparately(t *testing.T) {
	root := newRoot(t)
	for _, o := range []Outcome{Applied, Applied, RefusedApply, CheckNotRun, FailedCheck, RefusedParse} {
		_ = Record(root, o)
	}

	got, _ := Load(root)
	for name, want := range map[string]int{
		"applied": 2, "refused_apply": 1, "check_not_run": 1,
		"failed_check": 1, "refused_parse": 1,
	} {
		if got[name] != want {
			t.Errorf("%s = %d, want %d", name, got[name], want)
		}
	}
	if got.Plans() != 6 {
		t.Errorf("Plans() = %d, want 6 — the denominator counts plans, not counters", got.Plans())
	}
}

// FAIL OPEN. A corrupt tally is discarded, never repaired and never surfaced as
// an error — a caller's next move on "the tally is broken" is the same as on
// "there is no tally yet".
func TestAnUnreadableTallyFailsOpen(t *testing.T) {
	root := newRoot(t)
	if err := Record(root, Applied); err != nil {
		t.Fatal(err)
	}
	p := tallyPath(t, root)
	if p == "" {
		t.Fatal("no tally was written, so this test would pass without exercising anything")
	}
	if err := os.WriteFile(p, []byte("\x00\x01 not a tally\nrefused_parse notanumber\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Load(root)
	if err != nil {
		t.Errorf("a corrupt tally returned an error: %v — Load must fail open", err)
	}
	if len(got) != 0 {
		t.Errorf("garbage was parsed into %v, want an empty tally", got)
	}
	// And recording after corruption still works, rather than propagating it.
	if err := Record(root, Applied); err != nil {
		t.Errorf("Record after a corrupt tally returned %v, want nil", err)
	}
}

// Record may never fail a write. An unwritable state directory is the cheapest
// way to prove it: the tool must carry on regardless.
func TestRecordNeverFailsAWrite(t *testing.T) {
	root := newRoot(t)
	// A state home that is a FILE, not a directory: MkdirAll under it cannot
	// succeed, so state.Path errors and Record takes its failure path. (A NUL
	// in the value would be rejected by t.Setenv itself and prove nothing.)
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", blocker)
	if err := Record(root, Applied); err != nil {
		t.Errorf("Record returned %v with an unusable state home; it must never fail a write", err)
	}
}

// ⚠ THE BOUNDARY. This test is ADR-009's Enforced-by.
//
// The tally may hold counts and vocabulary names, and nothing else — no plan
// text, no paths, no anchors, no SHAs. The signature already makes it hard
// (Record takes two typed enums and cannot be handed a string), but a type is
// only a boundary while nobody adds a field, so this reads the written BYTES
// and refuses anything that is not `name count`.
func TestTheTallyNeverRecordsPlanContentOrPaths(t *testing.T) {
	root := newRoot(t)
	// Everything the vocabulary can produce, so the file is as full as it gets.
	for _, o := range []Outcome{Applied, RefusedParse, RefusedApply, CheckNotRun, FailedCheck} {
		_ = Record(root, o)
	}

	p := tallyPath(t, root)
	if p == "" {
		t.Fatal("no tally was written, so this test would pass without exercising anything")
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)

	line := regexp.MustCompile(`^[a-z_]+ [0-9]+$`)
	for _, l := range strings.Split(strings.TrimRight(body, "\n"), "\n") {
		if !line.MatchString(l) {
			t.Errorf("tally line %q is not `name count` — something other than a counter reached the file", l)
			continue
		}
		name := strings.SplitN(l, " ", 2)[0]
		if !names[name] {
			t.Errorf("tally holds %q, which is outside the closed vocabulary", name)
		}
	}

	// And explicitly: none of the things a plan is made of may appear. These are
	// the strings a future "just record the path" change would introduce.
	for _, forbidden := range []string{
		"/", "\\", "@@", ".go", ".txt", "anchor", "sha", "replace", "insert", "delete", "create",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("tally contains %q — a plan fragment, path or address reached disk:\n%s", forbidden, body)
		}
	}
}
