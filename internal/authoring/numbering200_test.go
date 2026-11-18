package authoring

import (
	"os"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// ADR-200 T1. A landing adds one to the landings, one to the candidate
// landings when it had a candidate, and one to the would-refuse landings when
// one of them was unshown: a write with two candidates is still one landing.
func TestNumberingCountsALandedWriteOnce(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if n := LoadNumbering(root); n != (Numbering{}) {
		t.Fatalf("nothing recorded yet: %+v, want zeros", n)
	}
	steps := []struct {
		candidates, unshown int
		want                Numbering
	}{
		{2, 2, Numbering{Landed: 1, Candidates: 1, WouldRefuse: 1}},
		{2, 0, Numbering{Landed: 2, Candidates: 2, WouldRefuse: 1}},
		{0, 0, Numbering{Landed: 3, Candidates: 2, WouldRefuse: 1}},
	}
	for _, s := range steps {
		if err := RecordNumbering(root, s.candidates, s.unshown); err != nil {
			t.Fatal(err)
		}
		if got := LoadNumbering(root); got != s.want {
			t.Errorf("after (%d, %d): %+v, want %+v", s.candidates, s.unshown, got, s.want)
		}
	}
	// Another root keeps its own counters.
	if n := LoadNumbering(t.TempDir()); n != (Numbering{}) {
		t.Errorf("another root reads %+v, want zeros", n)
	}
	// A file that is not counters reads as zero, and the next landing starts over.
	path, err := state.Path(root, "numbering")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("garbage\n\x00\xff landed_writes x\nlanded_writes -3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if n := LoadNumbering(root); n != (Numbering{}) {
		t.Errorf("a garbage file reads %+v, want zeros", n)
	}
	if err := RecordNumbering(root, 1, 0); err != nil {
		t.Fatal(err)
	}
	if n := LoadNumbering(root); n != (Numbering{Landed: 1, Candidates: 1}) {
		t.Errorf("after garbage: %+v, want one landing and one candidate", n)
	}
}
