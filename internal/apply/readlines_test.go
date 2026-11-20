package apply

import "github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"

// readLines is readLinesIn for a path that is not under any root, which the
// tests that load a FIFO or a file by absolute name need.
func readLines(path string) (t text, existed bool, err error) {
	f, _, openErr := regular.Open(path)
	return readLinesFrom(f, openErr)
}
