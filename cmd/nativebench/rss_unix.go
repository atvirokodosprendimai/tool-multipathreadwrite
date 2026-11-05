//go:build !windows

package main

import (
	"os"
	"runtime"
	"syscall"
)

// maxRSS is the run's peak resident set in bytes, from rusage: macOS reports
// bytes, Linux KiB.
func maxRSS(ps *os.ProcessState) int64 {
	ru, ok := ps.SysUsage().(*syscall.Rusage)
	if !ok {
		return 0
	}
	if runtime.GOOS == "darwin" {
		return ru.Maxrss
	}
	return ru.Maxrss * 1024
}
