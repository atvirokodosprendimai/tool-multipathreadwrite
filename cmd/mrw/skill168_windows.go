//go:build windows

package main

import (
	"errors"
	"syscall"
)

// heldByAnother reports whether err is Windows refusing a file another process
// holds — a sharing or a lock violation — which is the target's cause (ADR-125).
func heldByAnother(err error) bool {
	return errors.Is(err, syscall.Errno(32)) || errors.Is(err, syscall.Errno(33))
}
