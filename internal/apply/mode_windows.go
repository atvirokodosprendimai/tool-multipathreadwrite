//go:build windows

package apply

import "os"

// newFileMode is the mode of a file a plan creates. Windows has no umask, and
// only the read-only attribute follows these bits, so it stays 0644 (#714).
var newFileMode = os.FileMode(0o644)
