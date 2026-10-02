package writer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// flowTree is a checkout holding a.go, known to the ledger as if read, and
// the given harness.
func flowTree(t *testing.T, harness string) string {
	t.Helper()
	root := checkout(t)
	body := []byte("package a\nfunc A() {}\n")
	files := map[string][]byte{"a.go": body}
	if harness != "" {
		files[".quality-harness.json"] = []byte(harness)
	}
	for n, b := range files {
		if err := os.WriteFile(filepath.Join(root, n), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := seen.Record(root, map[string]seen.Observation{"a.go": {SHA: seen.SHA(body)}}); err != nil {
		t.Fatal(err)
	}
	return root
}

var goEdit = []apply.Input{{Path: "a.go", Start: 2, End: 2, Op: "replace", Body: []string{"func A() { _ = 1 }"}, Lines: -1, SrcLine: 1}}

// ADR-113 T1. The write sequence both surfaces share counts each outcome
// exactly once, in the phase that decides it: a refused harness in Prepare, a
// landing in Land, and the check's verdict in Verify, moved from applied.
func TestLandCountsEachOutcomeOnce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the checks are POSIX shell lines")
	}
	count := func(t *testing.T, root string) authoring.Tally {
		t.Helper()
		tally, err := authoring.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		return tally
	}
	run := func(t *testing.T, req Request) (Verified, error) {
		t.Helper()
		p, err := Prepare(req)
		if err != nil {
			return Verified{}, err
		}
		l, err := p.Land()
		if err != nil || l.Err != nil || l.LedgerErr != nil {
			return Verified{}, errors.Join(err, l.Err, l.LedgerErr)
		}
		v := l.Verify(context.Background())
		if v.CheckErr == nil {
			l.Settle(v)
		}
		return v, nil
	}

	t.Run("a malformed harness is refused before anything is written", func(t *testing.T) {
		root := flowTree(t, "{")
		_, err := run(t, Request{Root: root, In: goEdit, Opts: apply.Options{}})
		var r *Refusal
		if !errors.As(err, &r) || r.Stage != StageHarness {
			t.Fatalf("err %v, want a StageHarness refusal", err)
		}
		if got := count(t, root); got["refused_apply"] != 1 || got.Plans() != 1 {
			t.Errorf("tally %v, want refused_apply 1 and nothing else", got)
		}
		if b, _ := os.ReadFile(filepath.Join(root, "a.go")); string(b) != "package a\nfunc A() {}\n" {
			t.Errorf("a.go changed: %q", b)
		}
	})
	t.Run("a landing with no check is applied", func(t *testing.T) {
		root := flowTree(t, `{"check":"exit 3"}`)
		v, err := run(t, Request{Root: root, In: goEdit, Check: CheckOff})
		if err != nil || v.Check != nil {
			t.Fatalf("err %v check %+v, want a landing and no check", err, v.Check)
		}
		if got := count(t, root); got["applied"] != 1 || got.Plans() != 1 {
			t.Errorf("tally %v, want applied 1", got)
		}
		if p := authoring.LoadPricing(root); p.Candidates != 1 || p.Unchecked != 0 {
			t.Errorf("pricing %+v, want one candidate", p)
		}
	})
	t.Run("a failed check moves the landing to failed_check", func(t *testing.T) {
		root := flowTree(t, `{"check":"exit 3"}`)
		v, err := run(t, Request{Root: root, In: goEdit})
		if err != nil || v.Check == nil || !v.Check.Ran || v.Check.ExitCode != 3 {
			t.Fatalf("err %v check %+v, want a check that ran and exited 3", err, v.Check)
		}
		if got := count(t, root); got["failed_check"] != 1 || got.Plans() != 1 {
			t.Errorf("tally %v, want failed_check 1 and nothing else", got)
		}
	})
	t.Run("a check that cannot run moves the landing to check_not_run", func(t *testing.T) {
		root := flowTree(t, `{"check":"exit 0"}`)
		t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))
		v, err := run(t, Request{Root: root, In: goEdit})
		if err != nil || v.CheckErr == nil {
			t.Fatalf("err %v checkErr %v, want a landing whose check could not run", err, v.CheckErr)
		}
		if got := count(t, root); got["check_not_run"] != 1 || got.Plans() != 1 {
			t.Errorf("tally %v, want check_not_run 1 and nothing else", got)
		}
	})
	t.Run("an unread line is refused_apply", func(t *testing.T) {
		root := checkout(t)
		if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\nfunc A() {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		p, err := Prepare(Request{Root: root, In: goEdit, Check: CheckOff})
		if err != nil {
			t.Fatal(err)
		}
		l, err := p.Land()
		if err != nil || l.Res.Failed != 1 {
			t.Fatalf("err %v failed %d, want one failed hunk", err, l.Res.Failed)
		}
		if got := count(t, root); got["refused_apply"] != 1 || got.Plans() != 1 {
			t.Errorf("tally %v, want refused_apply 1", got)
		}
	})
	t.Run("a dry run counts nothing", func(t *testing.T) {
		root := flowTree(t, `{"check":"exit 3"}`)
		if _, err := run(t, Request{Root: root, In: goEdit, Opts: apply.Options{DryRun: true}}); err != nil {
			t.Fatal(err)
		}
		if got := count(t, root); got.Plans() != 0 {
			t.Errorf("tally %v, want nothing", got)
		}
	})
}

// A rename's source is gone, so its directory is what the check is scoped to,
// and its extension still says code was touched: a .go renamed to .txt
// removed code (moved from cmd/mrw by ADR-113).
func TestCheckPathsKeepsRenameSourcePackage(t *testing.T) {
	paths, code := CheckPaths([]apply.FileResult{
		{Path: "pkg/a.go", Written: true, Removed: true, RenamedTo: "other/out.txt"},
		{Path: "other/out.txt", Written: true, Created: true},
	})
	if !code {
		t.Fatal("renaming a .go to .txt did not count as code")
	}
	want := map[string]bool{"pkg": false, "other/out.txt": false}
	for _, p := range paths {
		if _, ok := want[p]; !ok {
			t.Errorf("unexpected check path %q", p)
		}
		want[p] = true
	}
	for p, seen := range want {
		if !seen {
			t.Errorf("missing check path %q in %v", p, paths)
		}
	}
}
