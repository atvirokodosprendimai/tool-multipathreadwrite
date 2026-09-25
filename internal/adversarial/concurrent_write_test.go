package adversarial

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// ADR-075, through the built binary. Eight writers off one read, each
// replacing a different line: every one that exits 0 has its edit in the file,
// and every other is refused as stale. Before the write lock the later rename
// discarded earlier edits while every writer printed "applied" (BACKLOG, the
// v1.25.1 round: 45-53% lost). Three rounds, because a race can go right by
// luck; the unit test in internal/writer is the deterministic one.
func TestConcurrentWritersOffOneReadLoseNothing(t *testing.T) {
	bin := mrwBinary(t)
	state := t.TempDir()
	const lines, writers = 16, 8
	for round := 0; round < 3; round++ {
		root := t.TempDir()
		var orig []string
		for k := 1; k <= lines; k++ {
			orig = append(orig, fmt.Sprintf("line %d", k))
		}
		if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte(strings.Join(orig, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		run := func(stdin string, args ...string) (int, string, error) {
			cmd := exec.Command(bin, append([]string{"--root", root}, args...)...)
			cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+state)
			cmd.Stdin = strings.NewReader(stdin)
			out, err := cmd.CombinedOutput()
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				return ee.ExitCode(), string(out), nil
			}
			return 0, string(out), err
		}
		if code, out, err := run("", "read", "f.txt"); err != nil || code != 0 {
			t.Fatalf("read: %d %v\n%s", code, err, out)
		}
		codes, outs := make([]int, writers), make([]string, writers)
		errs := make([]error, writers)
		var wg sync.WaitGroup
		for j := 0; j < writers; j++ {
			wg.Add(1)
			go func(j int) {
				defer wg.Done()
				codes[j], outs[j], errs[j] = run(fmt.Sprintf("@@ f.txt %d replace\nwriter %d\n", j+1, j), "write", "--no-check", "-")
			}(j)
		}
		wg.Wait()
		b, err := os.ReadFile(filepath.Join(root, "f.txt"))
		if err != nil {
			t.Fatal(err)
		}
		final := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
		if len(final) != lines {
			t.Fatalf("round %d: %d lines, want %d", round, len(final), lines)
		}
		var lost []int
		for j := 0; j < writers; j++ {
			if errs[j] != nil {
				t.Fatalf("writer %d did not run: %v", j, errs[j])
			}
			switch codes[j] {
			case 0:
				if final[j] != fmt.Sprintf("writer %d", j) {
					lost = append(lost, j)
				}
			case 1:
				if !strings.Contains(outs[j], "changed since") {
					t.Errorf("round %d: writer %d exited 1 without naming the change:\n%s", round, j, outs[j])
				}
			default:
				t.Errorf("round %d: writer %d exited %d:\n%s", round, j, codes[j], outs[j])
			}
		}
		if len(lost) > 0 {
			t.Fatalf("round %d: writer(s) %v exited 0 and their edit is gone: %q", round, lost, final)
		}
	}
}
