package writer

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// ADR-205 Decision 5. CheckAuto, once the zero value, ran the check on a code
// write; a Request that names no mode now runs nothing, so a surface that
// forgets to choose cannot start a check nobody asked for.
func TestARequestThatSetsNoModeRunsNoCheck(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the checks are POSIX shell lines")
	}
	root := flowTree(t, `{"check":"touch marker"}`)
	p, err := Prepare(Request{Root: root, In: goEdit})
	if err != nil {
		t.Fatal(err)
	}
	l, err := p.Land()
	if err != nil || l.Err != nil || !l.Res.Applied {
		t.Fatalf("err %v land %v applied %v, want a landing", err, l.Err, l.Res.Applied)
	}
	if l.CheckDue() {
		t.Errorf("a Request with no mode has a check due")
	}
	if v := l.Verify(context.Background()); v.Check != nil || v.CheckErr != nil {
		t.Errorf("check %+v err %v, want none", v.Check, v.CheckErr)
	}
	if _, err := os.Stat(filepath.Join(root, "marker")); err == nil {
		t.Errorf("the check ran: its marker exists")
	}
	if CheckOff != 0 {
		t.Errorf("CheckOff is %d, want the zero value", CheckOff)
	}
}
