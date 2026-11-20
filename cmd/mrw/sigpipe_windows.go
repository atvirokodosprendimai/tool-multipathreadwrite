//go:build windows

package main

// catchSIGPIPE does nothing on Windows, which has no SIGPIPE.
func catchSIGPIPE() {}
