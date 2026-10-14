//go:build !windows

package apply

import "os"

// lstatEntry is os.Lstat for an entry the directory lists. Off Windows every
// listed name can be opened as listed.
func lstatEntry(p string) (os.FileInfo, error) { return os.Lstat(p) }
