package read

import "github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"

// readCapped is readCappedIn for a path that is not under any root, which the
// tests that read /dev/zero, a FIFO or /proc need.
func readCapped(path string) ([]byte, error) {
	f, fi, err := regular.Open(path)
	if err != nil {
		return nil, err
	}
	return readCappedFile(f, fi)
}
