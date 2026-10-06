package writer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-127. Every write that lands moves the checkout's write counter, under
// the write lock; a refused write and a dry run do not.
func TestEveryLandedWriteBumpsTheCounter(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	for _, n := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(root, n), []byte(n+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	edit := func(name, body string, opt apply.Options) {
		t.Helper()
		opt.Force = true
		if _, err := Apply(root, []apply.Input{{Path: name, Start: 1, End: 1, Op: "replace", Body: []string{body}, Lines: -1}}, opt); err != nil {
			t.Fatal(err)
		}
	}
	start := Writes(root)
	edit("a.txt", "A", apply.Options{})
	edit("b.txt", "B", apply.Options{})
	if got := Writes(root) - start; got != 2 {
		t.Errorf("two landed writes moved the counter by %d, want 2", got)
	}
	edit("a.txt", "AA", apply.Options{DryRun: true})
	// Refused on its hunk: nothing lands, so nothing is counted.
	_, _ = Apply(root, []apply.Input{{Path: "a.txt", Start: 99, End: 99, Op: "replace", Body: []string{"x"}, Lines: -1}}, apply.Options{Force: true})
	if got := Writes(root) - start; got != 2 {
		t.Errorf("a dry run or a refused write moved the counter: %d, want 2", got)
	}
}

// ADR-127. A write's check runs after the write lock is released (ADR-075
// §5), so another writer can land while it runs; ADR-112's drift named only
// the files this write touched. The check here waits on a gate while a second
// write lands on another file; the first write's Verify counts it.
func TestAWriteThatLandsDuringTheCheckIsCounted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the check is a POSIX shell line")
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	gate := t.TempDir()
	started, release := filepath.Join(gate, "started"), filepath.Join(gate, "go")
	root := flowTree(t, fmt.Sprintf(`{"check":"touch %s; i=0; while [ ! -e %s ] && [ $i -lt 200 ]; do sleep 0.05; i=$((i+1)); done"}`, started, release))
	if err := os.WriteFile(filepath.Join(root, "other.txt"), []byte("o\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := seen.Record(root, map[string]seen.Observation{"other.txt": {SHA: seen.SHA([]byte("o\n"))}}); err != nil {
		t.Fatal(err)
	}

	verify := func(other bool) Verified {
		t.Helper()
		_ = os.Remove(started)
		_ = os.Remove(release)
		p, err := Prepare(Request{Root: root, In: goEdit})
		if err != nil {
			t.Fatal(err)
		}
		l, err := p.Land()
		if err != nil || l.Err != nil {
			t.Fatalf("land: %v %v", err, l.Err)
		}
		done := make(chan Verified, 1)
		go func() { done <- l.Verify(context.Background()) }()
		for i := 0; i < 200; i++ {
			if _, err := os.Stat(started); err == nil {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		if other {
			if _, err := Apply(root, []apply.Input{{Path: "other.txt", Start: 1, End: 1, Op: "replace", Body: []string{"changed"}, Lines: -1}}, apply.Options{Force: true}); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(release, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		return <-done
	}
	if v := verify(true); v.Check == nil || !v.Check.Ran || v.DriftWriters != 1 {
		t.Errorf("a write landed during the check was not counted: %+v", v)
	}
	if v := verify(false); v.DriftWriters != 0 {
		t.Errorf("with no other write, DriftWriters is %d", v.DriftWriters)
	}
}
