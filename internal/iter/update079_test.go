package iter

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

// ADR-079. Two `mrw iter add` racing each other read the set, added and wrote
// it back, and the later write lost the earlier entry — or read the file while
// it was being rewritten, found it empty and wiped the set. Updates take turns:
// every entry survives.
func TestConcurrentUpdatesKeepEveryEntry(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	const n = 24
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := Update(root, func(s *Set) error { s.Add(fmt.Sprintf("f%02d.go", i)); return nil }); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	s, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Entries) != n {
		t.Errorf("%d entries survived %d racing updates", len(s.Entries), n)
	}
}

// ADR-079. An Update whose change fails writes nothing and returns the error as
// it is, so a refusal inside `mrw iter add` keeps its own exit status.
func TestAnUpdateThatFailsWritesNothing(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if _, err := Update(root, func(s *Set) error { s.Add("a.go"); return nil }); err != nil {
		t.Fatal(err)
	}
	boom := errors.New("refused")
	if _, err := Update(root, func(s *Set) error { s.Add("b.go"); return boom }); err != boom {
		t.Fatalf("Update returned %v, want the change's own error", err)
	}
	if s, _ := Load(root); len(s.Entries) != 1 {
		t.Errorf("a failed Update wrote: %v", s.Entries)
	}
}
