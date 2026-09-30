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
	settled := func() int {
		// Goroutines that are ending need a moment to be reaped.
		n := runtime.NumGoroutine()
		for i := 0; i < 50 && n > 0; i++ {
			time.Sleep(10 * time.Millisecond)
			if m := runtime.NumGoroutine(); m >= n {
				break
			} else {
				n = m
			}
		}
		return n
	}
	before := settled()

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
	if after := settled(); after > before {
		t.Errorf("%d goroutine(s) before, %d after: a capture reader was left blocked", before, after)
	}
}
