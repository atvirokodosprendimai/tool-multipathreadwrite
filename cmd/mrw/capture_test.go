package main

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"
)

// captureStdout puts os.Stdout back and ends its reader goroutine even when
// the command it runs panics or calls runtime.Goexit, as t.Fatal does; before,
// both skipped the cleanup, leaving stdout redirected into a pipe nobody read
// and the reader blocked for good (the Codex review of #295).
func TestCaptureStdoutCleansUpWhenTheCommandAborts(t *testing.T) {
	orig := os.Stdout
	before := runtime.NumGoroutine()
	// waitBack polls until the goroutine count is back to before, for up to
	// five seconds: a goroutine that has sent or closed may not have exited
	// yet, so one plateau proves nothing, while a leaked reader stays blocked
	// and keeps the count above before for good (the Codex review of #295).
	waitBack := func() int {
		n := runtime.NumGoroutine()
		for deadline := time.Now().Add(5 * time.Second); n > before && time.Now().Before(deadline); n = runtime.NumGoroutine() {
			time.Sleep(10 * time.Millisecond)
		}
		return n
	}
	func() {
		defer func() { _ = recover() }()
		_, _ = captureStdout(t, func() error {
			fmt.Print("partial")
			panic("boom")
		})
	}()
	if os.Stdout != orig {
		t.Fatal("after a panic os.Stdout still points at the capture pipe")
	}

	exited := make(chan struct{})
	go func() {
		defer close(exited)
		_, _ = captureStdout(t, func() error {
			fmt.Print("partial")
			runtime.Goexit()
			return nil
		})
	}()
	<-exited
	if os.Stdout != orig {
		t.Fatal("after runtime.Goexit os.Stdout still points at the capture pipe")
	}
	if after := waitBack(); after > before {
		t.Errorf("%d goroutine(s) before, %d five seconds after: a capture reader was left blocked", before, after)
	}
}
