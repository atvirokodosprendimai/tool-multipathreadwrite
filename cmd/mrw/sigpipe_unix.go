//go:build !windows

package main

import (
	"os"
	"os/signal"
	"syscall"
)

// catchSIGPIPE makes a write to a broken standard output return EPIPE instead of
// ending the process (#713). A Go program that does not call Notify for SIGPIPE
// dies on a broken pipe at fd 1 or 2, and a write that had already landed then
// lost its check, its steps and its tally. Notify, not Ignore: an ignored signal
// is inherited by the check's and the steps' children, a caught one is reset.
func catchSIGPIPE() {
	signal.Notify(make(chan os.Signal, 1), syscall.SIGPIPE)
}
