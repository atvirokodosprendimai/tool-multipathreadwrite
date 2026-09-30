package writer

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-102 T2. A partial commit wrote files mrw produced, so they are wholly
// known (ADR-005) — but writer.Apply recorded the ledger only when the plan
// applied whole, and the next write to them was refused as changed since read.
// The partial commit is forced without a seam: content renames commit before
// path ops, and an unlink in a read-only directory fails at commit.
func TestAPartialCommitRecordsWhatLanded(t *testing.T) {
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
	if err := os.Chmod(filepath.Join(root, "d"), 0o555); err != nil {
		t.Fatal(err)
	}
	d := filepath.Join(root, "d")
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
	if err == nil || res.Applied || errors.As(err, &le) || MutationOf(res) != Partial {
		t.Fatalf("want a partial commit with its commit error: err %v, applied %v, mutation %v", err, res.Applied, MutationOf(res))
	}
	l, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if o := l["a.txt"]; o.SHA != seen.SHA([]byte("A\n")) || !o.Whole() {
		t.Errorf("the file the partial commit wrote is not recorded wholly: %+v", o)
	}
	if o := l["d/x.txt"]; o.SHA != seen.SHA([]byte("x\n")) {
		t.Errorf("the file the partial commit did not touch lost its licence: %+v", o)
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
	res, err = Apply(root, []apply.Input{
		{Path: "c.txt", Op: "unlink", Lines: -1, SrcLine: 1, Index: 0},
		{Path: "b.txt", Op: "rename", Body: []string{"c.txt"}, Lines: -1, SrcLine: 2, Index: 1},
	}, apply.Options{Seen: ledger})
	if err != nil || !res.Applied {
		t.Fatalf("unlink c + rename b→c did not apply: %v %+v", err, res.Hunks)
	}
	if l, err = seen.Load(root); err != nil {
		t.Fatal(err)
	}
	if o, ok := l["c.txt"]; !ok || o.SHA != seen.SHA([]byte("b\n")) || !o.Whole() {
		t.Errorf("c.txt, on disk with b's body, is not wholly known: %+v (present %v)", o, ok)
	}
}

// ADR-102 T2. A partial commit whose ledger also cannot be saved keeps its
// commit error, with the ledger's failure added: a LedgerError here would be
// counted as a clean landing by both callers.
func TestAPartialCommitWhoseLedgerFailsKeepsItsCommitError(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("a read-only file or directory does not stop this user from writing it")
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
	ledger, err := seen.Snapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	d := filepath.Join(root, "d")
	led, err := seen.ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []struct {
		path string
		mode os.FileMode
	}{{d, 0o555}, {led, 0o444}} {
		if err := os.Chmod(p.path, p.mode); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { _ = os.Chmod(d, 0o755); _ = os.Chmod(led, 0o600) })
	res, err := Apply(root, []apply.Input{
		{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, SrcLine: 1, Index: 0},
		{Path: "d/x.txt", Op: "unlink", Lines: -1, SrcLine: 3, Index: 1},
	}, apply.Options{Seen: ledger})
	var le *LedgerError
	if err == nil || errors.As(err, &le) || MutationOf(res) != Partial || !strings.Contains(err.Error(), "the ledger could not record what landed") {
		t.Fatalf("want the commit error with the ledger failure added, not a LedgerError: err %v, mutation %v", err, MutationOf(res))
	}
}
