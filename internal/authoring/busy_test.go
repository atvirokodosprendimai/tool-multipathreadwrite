package authoring

import (
	"io/fs"
	"os"
	"reflect"
	"syscall"
	"testing"
)

// On Linux a regular file under another process's lease answers EWOULDBLOCK to
// the non-blocking open the counters are read with. Read as "no counters yet",
// the save that follows would overwrite the history with one landing. A file
// that stays busy is left as it is: the count of that write is lost, the
// history is not.
func TestACountersFileThatStaysBusyIsNotOverwritten(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	record := func() {
		Record(root, Applied)
		RecordRecent(root, 1)
		RecordPricing(root, true, true, PricedHeld)
		RecordNumbering(root, 1, 1)
	}
	type all struct {
		t Tally
		r int
		p Pricing
		n Numbering
	}
	load := func() all {
		tally, _ := Load(root)
		return all{tally, len(Recent(root)), LoadPricing(root), LoadNumbering(root)}
	}
	// Two of each: a save from zero would write one, which must not look like the history.
	record()
	record()
	before := load()
	if before.t.Plans() != 2 || before.r != 2 || before.p.Candidates != 2 || before.n.Landed != 2 {
		t.Fatalf("the fixture did not record two of each: %+v", before)
	}

	real := openRegular
	openRegular = func(string) (*os.File, fs.FileInfo, error) { return nil, nil, syscall.EWOULDBLOCK }
	record()
	openRegular = real

	if after := load(); !reflect.DeepEqual(after, before) {
		t.Errorf("a busy counters file was overwritten or counted: %+v, want %+v", after, before)
	}
	record()
	if after := load(); after.t.Plans() != 3 || after.r != 3 || after.p.Candidates != 3 || after.n.Landed != 3 {
		t.Errorf("once the file opens again the counts go on from the history: %+v", after)
	}
}
