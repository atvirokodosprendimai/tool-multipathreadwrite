//go:build unix

package regular

import (
	"os"
	"syscall"
)

// Release opens pipe for writing without blocking and closes it, so a reader a
// mutant left blocked on its open returns and the test can end.
func Release(pipe string) {
	if w, err := os.OpenFile(pipe, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
		_ = w.Close()
	}
}
