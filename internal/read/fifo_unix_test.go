//go:build unix

package read

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// ADR-074 T2. A named FIFO hung `read`, and `--stat`, and a symlink to one:
// os.ReadFile blocks until something writes to the pipe. The walk has refused a
// non-regular candidate since ADR-007; a named spec is reported the same way,
// a directory keeps its own "is a directory", and the regular file named beside
// them is still served.
func TestAFIFOIsReportedByNameNotWaitedOn(t *testing.T) {
	root := t.TempDir()
	fifo := filepath.Join(root, "p")
	if err := syscall.Mkfifo(fifo, 0o644); err != nil {
		t.Skipf("no FIFO here: %v", err)
	}
	if err := os.Symlink("p", filepath.Join(root, "lp")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, stat := range []bool{false, true} {
		var specs []Spec
		for _, s := range []string{"p", "lp", "d", "a.go"} {
			sp, err := ParseSpec(s)
			if err != nil {
				t.Fatal(err)
			}
			specs = append(specs, sp)
		}
		type result struct {
			out      string
			problems int
		}
		done := make(chan result, 1)
		go func() {
			var b bytes.Buffer
			_, problems := Run(&b, root, specs, Options{Numbers: true, Stat: stat})
			done <- result{b.String(), problems}
		}()
		select {
		case <-time.After(3 * time.Second):
			// Let the stuck open return, so the goroutine does not outlive the test.
			if f, err := os.OpenFile(fifo, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
				f.Close()
			}
			t.Fatalf("read (stat=%v) is still waiting on the FIFO at 3 s", stat)
		case got := <-done:
			for _, want := range []string{
				"==> p  UNREADABLE  not a regular file",
				"==> lp  UNREADABLE  not a regular file",
				"==> d  UNREADABLE",
				"is a directory",
				"==> a.go  1L",
			} {
				if !strings.Contains(got.out, want) {
					t.Errorf("stat=%v: the read does not say %q:\n%s", stat, want, got.out)
				}
			}
			if strings.Contains(got.out, "==> d  UNREADABLE  not a regular file") {
				t.Errorf("stat=%v: a directory is reported as a pipe or device:\n%s", stat, got.out)
			}
			if got.problems != 3 {
				t.Errorf("stat=%v: %d problem(s), want 3:\n%s", stat, got.problems, got.out)
			}
		}
	}
}
