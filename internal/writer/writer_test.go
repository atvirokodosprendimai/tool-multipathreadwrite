package writer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

func checkout(t *testing.T) string {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	return t.TempDir()
}

// TestNoTwoWritersAreInsideAtOnce is ADR-075's Enforced-by. Each writer that
// gets in lingers long enough for every other to arrive, so without the lock
// they pile in and the peak says so, whatever order the scheduler picks.
func TestNoTwoWritersAreInsideAtOnce(t *testing.T) {
	root := checkout(t)
	var mu sync.Mutex
	in, peak := 0, 0
	inside = func() {
		mu.Lock()
		in++
		if in > peak {
			peak = in
		}
		mu.Unlock()
		time.Sleep(100 * time.Millisecond)
		mu.Lock()
		in--
		mu.Unlock()
	}
	t.Cleanup(func() { inside = nil })

	const n = 6
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res, err := Apply(root, []apply.Input{{Path: fmt.Sprintf("f%d.txt", i), Op: "create", Body: []string{"x"}, Lines: -1, SrcLine: 1}}, apply.Options{})
			if err == nil && !res.Applied {
				err = fmt.Errorf("writer %d did not apply: %+v", i, res.Hunks)
			}
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if peak != 1 {
		t.Fatalf("%d writers were inside at once, want 1", peak)
	}
}

// ADR-075. Two writers off one read: the first lands, and the second, whose
// file changed while it waited, is refused rather than rebased onto the first
// writer's result. Apply validates against the ledger the CALLER loaded before
// the lock; reloading it inside would hand the second writer the first
// writer's whole-file licence.
func TestAWriterWhoseFileChangedWhileItWaitedIsRefused(t *testing.T) {
	root := checkout(t)
	body := []byte("one\ntwo\nthree\n")
	if err := os.WriteFile(filepath.Join(root, "f.txt"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := seen.Record(root, map[string]seen.Observation{"f.txt": {SHA: seen.SHA(body)}}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	a, err := Apply(root, []apply.Input{{Path: "f.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, SrcLine: 1}}, apply.Options{Seen: snapshot})
	if err != nil || !a.Applied {
		t.Fatalf("the first writer did not land: %v %+v", err, a.Hunks)
	}
	b, err := Apply(root, []apply.Input{{Path: "f.txt", Start: 3, End: 3, Op: "replace", Body: []string{"B"}, Lines: -1, SrcLine: 1}}, apply.Options{Seen: snapshot})
	if err != nil {
		t.Fatal(err)
	}
	if b.Applied || len(b.Hunks) == 0 || !strings.Contains(b.Hunks[0].Reason, "changed since") {
		t.Fatalf("the second writer was not refused as stale: %+v", b.Hunks)
	}
	got, _ := os.ReadFile(filepath.Join(root, "f.txt"))
	if string(got) != "A\ntwo\nthree\n" {
		t.Fatalf("the file holds %q", got)
	}
}

// ADR-075. What landed is in the ledger before the lock is released: a written
// file wholly, and an unlinked one not at all.
func TestWhatLandedIsRecordedBeforeTheLockIsReleased(t *testing.T) {
	root := checkout(t)
	for name, body := range map[string]string{"f.txt": "one\n", "g.txt": "gone\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := seen.Record(root, map[string]seen.Observation{"g.txt": {SHA: seen.SHA([]byte("gone\n"))}}); err != nil {
		t.Fatal(err)
	}
	res, err := Apply(root, []apply.Input{
		{Path: "f.txt", Start: 1, End: 1, Op: "replace", Body: []string{"two"}, Lines: -1, SrcLine: 1, Index: 0},
		{Path: "g.txt", Op: "unlink", Lines: -1, SrcLine: 3, Index: 1},
	}, apply.Options{})
	if err != nil || !res.Applied {
		t.Fatalf("did not land: %v %+v", err, res.Hunks)
	}
	l, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if o, ok := l["f.txt"]; !ok || o.SHA != seen.SHA([]byte("two\n")) || !o.Whole() {
		t.Fatalf("the written file is not recorded wholly: %+v", l["f.txt"])
	}
	if _, ok := l["g.txt"]; ok {
		t.Fatal("the unlinked file is still in the ledger")
	}
}
