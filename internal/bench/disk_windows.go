//go:build windows

package bench

import "errors"

// freeGiB is not built for Windows: bench runs on the operator's macOS or
// Linux machine (ADR-156), and a disk floor there cannot be checked.
func freeGiB(string) (float64, error) {
	return 0, errors.New("the disk floor is checked on macOS and Linux only")
}
