//go:build linux

package rooted

import (
	"os"
	"strconv"
)

// fdPath is the path the open descriptor f names now, as the system reports it
// (#727): the opened file itself says where it is, rather than a second lookup of
// a name.
func fdPath(f *os.File) (string, bool) {
	p, err := os.Readlink("/proc/self/fd/" + strconv.Itoa(int(f.Fd())))
	return p, err == nil && p != ""
}
