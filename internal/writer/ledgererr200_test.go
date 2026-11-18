package writer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-200 Decision 2: a commit whose ledger update failed is a landing, and
// Land counts it once like any other. The branch was named as driven by no
// test when T1 merged; this drives it.
func TestALandingWhoseLedgerUpdateFailsIsCountedOnce(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("a read-only file does not stop this user from writing it")
	}
	const body = "a\nb\nc\n"
	root := checkout(t)
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := seen.Record(root, map[string]seen.Observation{"f.txt": {SHA: seen.SHA([]byte(body)), Written: true}}); err != nil {
		t.Fatal(err)
	}
	led, err := seen.ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(led, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(led, 0o600) })
	p, err := Prepare(Request{Root: root, Check: CheckOff, In: []apply.Input{
		{Path: "f.txt", Start: 2, End: 2, Op: "replace", Body: []string{"B"}, Lines: -1, SrcLine: 1},
	}})
	if err != nil {
		t.Fatal(err)
	}
	l, err := p.Land()
	if err != nil || l.LedgerErr == nil || MutationOf(l.Res) == None {
		t.Fatalf("want a landing whose ledger update failed: err %v, ledger err %v, mutation %v", err, l.LedgerErr, MutationOf(l.Res))
	}
	if got, _ := os.ReadFile(filepath.Join(root, "f.txt")); string(got) != "a\nB\nc\n" {
		t.Fatalf("the write did not land: %q", got)
	}
	want := authoring.Numbering{Landed: 1, Candidates: 1, WouldRefuse: 1}
	if got := authoring.LoadNumbering(root); got != want {
		t.Errorf("after a landing whose ledger update failed: %+v, want %+v", got, want)
	}
}
