//go:build !windows

package main

// heldByAnother reports whether err is a file another process holds; only
// Windows refuses an open for that (ADR-125).
func heldByAnother(error) bool { return false }
