//go:build !windows

package apply

import "os"

// keepAttributes has nothing to carry outside Windows: a POSIX file's mode is
// kept by stageFile's chmod, and there are no Hidden or System attributes.
func keepAttributes(*tree, string, *os.File) error { return nil }
