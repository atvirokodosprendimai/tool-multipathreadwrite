package writer

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-102 T2. A partial commit wrote files mrw produced, so they are wholly
// known (ADR-005) — but writer.Apply recorded the ledger only when the plan
// applied whole, and the next write to them was refused as changed since read.
// Since ADR-209 a failed commit puts its content files back, and a partial one
// is left only where an undo step could not put a file back or no aside could be
// made: no filesystem here refuses a link on demand, so the partial Result is
// built, and ledgerChanges, the step that reads it, is what is tested.
func TestAPartialCommitRecordsWhatLanded(t *testing.T) {
	root := checkout(t)
	if err := seen.Record(root, map[string]seen.Observation{
		"a.txt": {SHA: seen.SHA([]byte("a\n"))}, "d/x.txt": {SHA: seen.SHA([]byte("x\n"))},
	}); err != nil {
		t.Fatal(err)
	}
	ledger, err := seen.Snapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	wrote, gone := ledgerChanges(root, apply.Result{Files: []apply.FileResult{
		{Path: "a.txt", Written: true, SHAAfter: seen.SHA([]byte("A\n"))},
		{Path: "b.txt", Written: true, SHAAfter: seen.SHA([]byte("B\n")), NotHeld: true},
		{Path: "c.txt", Written: false},
	}}, apply.Options{Seen: ledger})
	if o := wrote["a.txt"]; o.SHA != seen.SHA([]byte("A\n")) || !o.Whole() {
		t.Errorf("the file the partial commit wrote is not recorded wholly: %+v", o)
	}
	if _, ok := wrote["b.txt"]; ok {
		t.Errorf("b.txt, whose path no longer holds what mrw wrote, is recorded: %+v", wrote["b.txt"])
	}
	if _, ok := wrote["c.txt"]; ok || len(gone) != 0 {
		t.Errorf("an unwritten file is recorded %v or dropped %v", wrote, gone)
	}

	// Drop runs before Record: an unlink of c.txt and a rename of b.txt onto
	// c.txt in one plan leave two records for c.txt, and c.txt is on disk.
	root = checkout(t)
	for name, body := range map[string]string{"b.txt": "b\n", "c.txt": "c\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := seen.Record(root, map[string]seen.Observation{
		"b.txt": {SHA: seen.SHA([]byte("b\n"))}, "c.txt": {SHA: seen.SHA([]byte("c\n"))},
	}); err != nil {
		t.Fatal(err)
	}
	if ledger, err = seen.Snapshot(root); err != nil {
		t.Fatal(err)
	}
	res, err := Apply(root, []apply.Input{
		{Path: "c.txt", Op: "unlink", Lines: -1, SrcLine: 1, Index: 0},
		{Path: "b.txt", Op: "rename", Body: []string{"c.txt"}, Lines: -1, SrcLine: 2, Index: 1},
	}, apply.Options{Seen: ledger})
	if err != nil || !res.Applied {
		t.Fatalf("unlink c + rename b→c did not apply: %v %+v", err, res.Hunks)
	}
	l, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if o, ok := l["c.txt"]; !ok || o.SHA != seen.SHA([]byte("b\n")) || !o.Whole() {
		t.Errorf("c.txt, on disk with b's body, is not wholly known: %+v (present %v)", o, ok)
	}
}

// ADR-209. A commit whose unlink is refused writes nothing: a.txt is put back, the
// write is not a mutation, and a.txt stays licensed by the read made before it.
func TestAFailedCommitPutsTheFileBackAndKeepsItsLicence(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("a read-only directory does not stop this user from writing it")
	}
	root := checkout(t)
	if err := os.MkdirAll(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"a.txt": "a\n", "d/x.txt": "x\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := seen.Record(root, map[string]seen.Observation{
		"a.txt": {SHA: seen.SHA([]byte("a\n"))}, "d/x.txt": {SHA: seen.SHA([]byte("x\n"))},
	}); err != nil {
		t.Fatal(err)
	}
	d := filepath.Join(root, "d")
	if err := os.Chmod(d, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(d, 0o755) })
	ledger, err := seen.Snapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Apply(root, []apply.Input{
		{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, SrcLine: 1, Index: 0},
		{Path: "d/x.txt", Op: "unlink", Lines: -1, SrcLine: 3, Index: 1},
	}, apply.Options{Seen: ledger})
	var le *LedgerError
	if err == nil || res.Applied || errors.As(err, &le) || MutationOf(res) != None {
		t.Fatalf("want a refused commit that wrote nothing: err %v, applied %v, mutation %v", err, res.Applied, MutationOf(res))
	}
	if got, _ := os.ReadFile(filepath.Join(root, "a.txt")); string(got) != "a\n" {
		t.Errorf("a.txt = %q, want it put back", got)
	}
	l, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if o := l["a.txt"]; o.SHA != seen.SHA([]byte("a\n")) {
		t.Errorf("a.txt lost the licence of the read made before the write: %+v", o)
	}
}

// ADR-102 T2. A partial commit whose ledger also cannot be saved keeps its
// commit error, with the ledger's failure added: a LedgerError here would be
// counted as a clean landing by both callers.
func TestAPartialCommitWhoseLedgerFailsKeepsItsCommitError(t *testing.T) {
	commit, lerr := errors.New("c.txt: rename refused (UNDO INCOMPLETE: a.txt)"), errors.New("the ledger is read-only")
	err := ledgerFailed(Partial, commit, lerr)
	var le *LedgerError
	if errors.As(err, &le) || !errors.Is(err, commit) || !strings.Contains(err.Error(), "the ledger could not record what landed") {
		t.Errorf("a partial commit: %v, want the commit error with the ledger failure added, not a LedgerError", err)
	}
	if err := ledgerFailed(Complete, commit, lerr); !errors.As(err, &le) {
		t.Errorf("a complete commit: %v, want a LedgerError", err)
	}
}

// partialApply makes applyFn answer as a commit does when an undo could not finish:
// a.go written (its path holds mrw's bytes), the unlink failed, an error.
func partialApply(t *testing.T, root string) {
	t.Helper()
	real := applyFn
	t.Cleanup(func() { applyFn = real })
	applyFn = func(r string, _ []apply.Input, _ apply.Options) (apply.Result, error) {
		if err := os.WriteFile(filepath.Join(r, "a.go"), []byte("package b\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return apply.Result{Root: r, Failed: 1, StrictSingleLine: 1, Files: []apply.FileResult{
			{Path: "a.go", Written: true, SHAAfter: seen.SHA([]byte("package b\n"))},
		}}, errors.New("d/x.txt: unlink refused (ALREADY WRITTEN: a.go); UNDO INCOMPLETE: a.go was written with no aside")
	}
}

func partialTree(t *testing.T) string {
	t.Helper()
	root := checkout(t)
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := seen.Record(root, map[string]seen.Observation{"a.go": {SHA: seen.SHA([]byte("package a\n"))}}); err != nil {
		t.Fatal(err)
	}
	return root
}

var partialIn = []apply.Input{{Path: "a.go", Start: 1, End: 1, Op: "replace", Body: []string{"package b"}, Lines: -1, SrcLine: 1}}

// ADR-102 T1 and T2, through the seam. A partial commit is a landed write: its file is
// wholly recorded in the ledger, and the call is counted partially_applied, landed,
// priced and put in the recent ring, by both surfaces (they share Prepare and Land).
func TestAPartialCommitIsRecordedCountedAndPricedThroughTheFlow(t *testing.T) {
	root := partialTree(t)
	partialApply(t, root)
	p, err := Prepare(Request{Root: root, In: partialIn, Check: CheckOff})
	if err != nil {
		t.Fatal(err)
	}
	l, err := p.Land()
	if err != nil || l.Err == nil || l.LedgerErr != nil {
		t.Fatalf("a partial commit: land %v, commit error %v, ledger error %v", err, l.Err, l.LedgerErr)
	}
	tally, err := authoring.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if tally["partially_applied"] != 1 || tally["refused_apply"] != 0 || tally.Landed() != 1 {
		t.Errorf("tally %v landed %d, want partially_applied 1, refused_apply 0, landed 1", tally, tally.Landed())
	}
	if n, pr := len(authoring.Recent(root)), authoring.LoadPricing(root); n != 1 || pr.Candidates != 1 {
		t.Errorf("recent ring %d, strict_candidates %d; want 1 and 1", n, pr.Candidates)
	}
	if o := mustLoad(t, root)["a.go"]; o.SHA != seen.SHA([]byte("package b\n")) || !o.Whole() {
		t.Errorf("the file the partial commit wrote is not recorded wholly: %+v", o)
	}
}

func mustLoad(t *testing.T, root string) map[string]seen.Observation {
	t.Helper()
	l, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

// ADR-102 T2, through the seam. The ledger of a partial commit is recorded by
// applyCounted itself, and the commit's error stays the error.
func TestAPartialCommitIsRecordedByApplyCounted(t *testing.T) {
	root := partialTree(t)
	partialApply(t, root)
	res, _, err := applyCounted(root, partialIn, apply.Options{})
	var le *LedgerError
	if err == nil || errors.As(err, &le) || MutationOf(res) != Partial {
		t.Fatalf("a partial commit: err %v, mutation %v", err, MutationOf(res))
	}
	if o := mustLoad(t, root)["a.go"]; o.SHA != seen.SHA([]byte("package b\n")) || !o.Whole() {
		t.Errorf("a.go is not recorded wholly: %+v", o)
	}
}

// ADR-102 T2, through the seam. A partial commit whose ledger cannot be saved keeps
// its commit error with the ledger's failure added, not a LedgerError, which both
// surfaces would count as a clean landing.
func TestAPartialCommitWhoseLedgerCannotBeSavedKeepsItsCommitErrorEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("a read-only file does not stop this user from writing it")
	}
	root := partialTree(t)
	partialApply(t, root)
	led, err := seen.ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(led, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(led, 0o600) })
	res, _, err := applyCounted(root, partialIn, apply.Options{})
	var le *LedgerError
	if err == nil || errors.As(err, &le) || MutationOf(res) != Partial || !strings.Contains(err.Error(), "the ledger could not record what landed") {
		t.Fatalf("want the commit error with the ledger failure added, not a LedgerError: err %v, mutation %v", err, MutationOf(res))
	}
}
