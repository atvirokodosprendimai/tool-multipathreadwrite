package apply

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ADR-107 T1. apply read a file a plan edits whole, with os.ReadFile, before
// its licence was checked, so a file of any size reached that allocation. A
// file over the limit is refused on its hunk at validation, naming its size
// and the limit, and writes nothing; readLines reads through a bound too, for
// a file that grows after its size was taken; a file under the limit edits as
// before.
func TestAFileOverTheEditLimitIsRefusedBeforeItIsRead(t *testing.T) {
	old := maxLoadBytes
	t.Cleanup(func() { maxLoadBytes = old })
	maxLoadBytes = 16
	root := t.TempDir()
	big := strings.Repeat("x\n", 20) // 40 bytes
	write(t, root, "big.txt", big)
	write(t, root, "small.txt", "a\n")

	res, err := Apply(root, []Input{{Path: "big.txt", Start: 1, End: 1, Op: "replace", Body: []string{"y"}, Lines: -1, Index: 0}}, Options{Force: true})
	h := hunkFor(t, res, "big.txt")
	if err != nil || res.Applied || h.Status != StatusFailed || !strings.Contains(h.Reason, "40 bytes") || !strings.Contains(h.Reason, "16") {
		t.Errorf("a file over the limit was not refused on its hunk naming size and limit: err %v, %+v", err, h)
	}
	if read(t, root, "big.txt") != big {
		t.Error("big.txt was written")
	}

	if _, _, err := readLines(filepath.Join(root, "big.txt")); err == nil || !strings.Contains(err.Error(), "16") {
		t.Errorf("readLines read a file over the limit whole: %v", err)
	}

	res, err = Apply(root, []Input{{Path: "small.txt", Start: 1, End: 1, Op: "replace", Body: []string{"b"}, Lines: -1, Index: 0}}, Options{Force: true})
	if err != nil || !res.Applied || read(t, root, "small.txt") != "b\n" {
		t.Errorf("a file under the limit did not edit as before: %v %+v", err, res)
	}

	// The Codex review of #302: a file that grows past the limit between its
	// stat and its read is refused on its hunk, with every verdict kept.
	write(t, root, "grow.txt", "g\n")
	real := loadFn
	t.Cleanup(func() { loadFn = real })
	loadFn = func(path string) (text, bool, error) {
		if filepath.Base(path) == "grow.txt" {
			if err := os.WriteFile(path, []byte(big), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return real(path)
	}
	res, err = Apply(root, []Input{
		{Path: "grow.txt", Start: 1, End: 1, Op: "replace", Body: []string{"G"}, Lines: -1, Index: 0},
		{Path: "small.txt", Start: 1, End: 1, Op: "replace", Body: []string{"c"}, Lines: -1, Index: 1},
	}, Options{Force: true})
	if g := hunkFor(t, res, "grow.txt"); err != nil || res.Applied || g.Status != StatusFailed || !strings.Contains(g.Reason, "grew past") {
		t.Errorf("a file grown past the limit was not refused on its hunk: err %v, %+v", err, g)
	}
	if s := hunkFor(t, res, "small.txt"); s.Status != StatusSkipped || read(t, root, "small.txt") != "b\n" {
		t.Errorf("the sibling was not skipped and left alone: %+v", s)
	}
	loadFn = real

	// And the read itself is bounded: a 4 MB file past a 16-byte limit costs
	// the limit, not the file (the Codex review of #302).
	huge := filepath.Join(root, "huge.txt")
	if err := os.WriteFile(huge, []byte(strings.Repeat("h", 4<<20)), 0o644); err != nil {
		t.Fatal(err)
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_, _, rerr := readLines(huge)
	runtime.ReadMemStats(&after)
	if rerr == nil {
		t.Error("readLines read a 4 MB file past a 16-byte limit")
	}
	if d := after.TotalAlloc - before.TotalAlloc; d > 256<<10 {
		t.Errorf("readLines allocated %d bytes for a file past the limit, want under 256 KB", d)
	}
}
