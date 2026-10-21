//go:build windows

package bench

import "errors"

// lockFile is not built for Windows: bench runs on the operator macOS or
// Linux machine (ADR-156), and the shims of the container run there.
func lockFile(string) (func(), error) {
	return nil, errors.New("the container lock is taken on macOS and Linux only")
}
