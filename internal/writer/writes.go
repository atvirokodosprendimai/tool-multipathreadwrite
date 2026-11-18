package writer

import (
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// writesName is the state file holding the checkout's write counter (ADR-127).
const writesName = "writes"

// readCounter is regular.ReadFile, as a variable so a test can hold the
// counter busy. The counter is read without blocking, never with a fallback to
// a blocking open (ADR-201).
var readCounter = regular.ReadFile

// Writes is how many writes have landed in root's checkout, as the counter in
// its state directory says (ADR-127). A counter that is missing or cannot be
// read is zero: the advisory it feeds is best effort, never a refusal.
func Writes(root string) int64 {
	n, _ := readWrites(root)
	return n
}

// readWrites is Writes with whether the counter could be relied on: a counter
// that is missing, is not a regular file or does not parse reads as zero and
// ok, while one that could not be read for another reason (a lease on Linux
// answers EWOULDBLOCK, or a permission) is not ok, so bumpWrites does not save
// over what it never saw (ADR-201).
func readWrites(root string) (n int64, ok bool) {
	top, _ := state.Checkout(root)
	p, err := state.Path(top, writesName)
	if err != nil {
		return 0, false
	}
	b, err := readCounter(p)
	if err != nil {
		return 0, errors.Is(err, fs.ErrNotExist) || errors.Is(err, regular.ErrNotRegular)
	}
	n, err = strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return 0, true
	}
	return n, true
}

// bumpWrites moves root's write counter on by one and returns it, or zero when
// it could not (ADR-127). It is called under the write lock, so no two writers
// bump at once.
func bumpWrites(root string) int64 {
	top, _ := state.Checkout(root)
	p, err := state.Path(top, writesName)
	if err != nil {
		return 0
	}
	n, ok := readWrites(root)
	if !ok {
		return 0
	}
	n++
	if err := state.Write(p, []byte(fmt.Sprintf("%d\n", n)), 0o600); err != nil {
		return 0
	}
	return n
}
