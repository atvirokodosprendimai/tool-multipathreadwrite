package main

import (
	"os"
	"testing"
	"time"
)

// The Codex review of #371, P2. A PATH no plan could name is a usage error
// before standard input is read: with stdin an open pipe nobody writes to, the
// call must still return promptly instead of waiting for EOF.
func TestAUsageErrorDoesNotWaitForStandardInput(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = w.Close(); _ = r.Close() })
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = old })
	for _, path := range []string{"", "/abs/x.txt", "a\nb.txt"} {
		done := make(chan int, 1)
		go func() {
			_, code := runIn(t, t.TempDir(), "write", "--no-check", "--create", path)
			done <- code
		}()
		select {
		case code := <-done:
			if code != 2 {
				t.Errorf("--create %q: exit %d, want 2", path, code)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("--create %q waited for standard input before refusing the path", path)
		}
	}
}
