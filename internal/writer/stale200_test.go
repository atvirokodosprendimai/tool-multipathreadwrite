package writer

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-200 T1. A landing is counted once, in Land, before its check runs: not
// again by Settle, and not at all for a dry run or a refusal.
func TestALandingIsCountedOnceInLand(t *testing.T) {
	const body = "package a\nfunc A() {}\nfunc B() {}\n"
	// a.go is known the way a write leaves it: wholly licensed, nothing shown.
	tree := func(t *testing.T, harness string) string {
		t.Helper()
		root := checkout(t)
		for n, b := range map[string]string{"a.go": body, ".quality-harness.json": harness} {
			if err := os.WriteFile(filepath.Join(root, n), []byte(b), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if err := seen.Record(root, map[string]seen.Observation{"a.go": {SHA: seen.SHA([]byte(body)), Written: true}}); err != nil {
			t.Fatal(err)
		}
		return root
	}
	numbered := []apply.Input{{Path: "a.go", Start: 2, End: 2, Op: "replace", Body: []string{"func A() { _ = 1 }"}, Lines: -1, SrcLine: 1}}
	anchored := []apply.Input{{Path: "a.go", Start: 2, End: 2, Op: "replace", Anchor: "func A", Body: []string{"func A() { _ = 1 }"}, Lines: -1, SrcLine: 1}}
	counts := func(root string) authoring.Numbering { return authoring.LoadNumbering(root) }

	t.Run("a landing is counted before its check and not again after it", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("the check is a POSIX shell line")
		}
		root := tree(t, `{"check":"exit 0"}`)
		p, err := Prepare(Request{Root: root, In: numbered})
		if err != nil {
			t.Fatal(err)
		}
		l, err := p.Land()
		if err != nil || !l.CheckDue() {
			t.Fatalf("err %v checkDue %v, want a landing whose check is due", err, l.CheckDue())
		}
		want := authoring.Numbering{Landed: 1, Candidates: 1, WouldRefuse: 1}
		if got := counts(root); got != want {
			t.Fatalf("after Land: %+v, want %+v", got, want)
		}
		v := l.Verify(context.Background())
		if v.Check == nil || !v.Check.Ran {
			t.Fatalf("the check did not run: %+v", v)
		}
		l.Settle(v)
		if got := counts(root); got != want {
			t.Errorf("after Verify and Settle: %+v, want %+v", got, want)
		}
		if tally, _ := authoring.Load(root); tally.Landed() != counts(root).Landed {
			t.Errorf("landed_writes %d, the tally's landed %d", counts(root).Landed, tally.Landed())
		}
	})
	t.Run("a landing whose check cannot start is counted", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("the check is a POSIX shell line")
		}
		root := tree(t, `{"check":"exit 0"}`)
		t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))
		p, err := Prepare(Request{Root: root, In: numbered})
		if err != nil {
			t.Fatal(err)
		}
		l, err := p.Land()
		if err != nil {
			t.Fatal(err)
		}
		if v := l.Verify(context.Background()); v.CheckErr == nil {
			t.Fatalf("want a check that could not start, got %+v", v)
		}
		if got := counts(root); got.Landed != 1 || got.Candidates != 1 {
			t.Errorf("%+v, want one landing and one candidate", got)
		}
	})
	t.Run("an anchored landing is a landing and no candidate", func(t *testing.T) {
		root := tree(t, `{"check":"exit 0"}`)
		p, _ := Prepare(Request{Root: root, In: anchored, Check: CheckOff})
		if _, err := p.Land(); err != nil {
			t.Fatal(err)
		}
		if got := counts(root); got != (authoring.Numbering{Landed: 1}) {
			t.Errorf("%+v, want one landing alone", got)
		}
	})
	t.Run("a dry run and a refusal count nothing", func(t *testing.T) {
		root := tree(t, `{"check":"exit 0"}`)
		p, _ := Prepare(Request{Root: root, In: numbered, Check: CheckOff, Opts: apply.Options{DryRun: true}})
		if _, err := p.Land(); err != nil {
			t.Fatal(err)
		}
		// Line 9 is out of range: the hunk fails and nothing lands.
		bad := []apply.Input{{Path: "a.go", Start: 9, End: 9, Op: "replace", Body: []string{"x"}, Lines: -1, SrcLine: 1}}
		p, _ = Prepare(Request{Root: root, In: bad, Check: CheckOff})
		if l, err := p.Land(); err != nil || l.Res.Failed != 1 {
			t.Fatalf("err %v failed %d, want one refused hunk", err, l.Res.Failed)
		}
		if got := counts(root); got != (authoring.Numbering{}) {
			t.Errorf("%+v, want zeros", got)
		}
	})
}
