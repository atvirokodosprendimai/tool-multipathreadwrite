package writer

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// writesName is the state file holding the checkout's write counter (ADR-127).
const writesName = "writes"

// Writes is how many writes have landed in root's checkout, as the counter in
// its state directory says (ADR-127). A counter that is missing or cannot be
// read is zero: the advisory it feeds is best effort, never a refusal.
func Writes(root string) int64 {
	p, err := state.Path(root, writesName)
	if err != nil {
		return 0
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return 0
	}
	n, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// bumpWrites moves root's write counter on by one and returns it, or zero when
// it could not (ADR-127). It is called under the write lock, so no two writers
// bump at once.
func bumpWrites(root string) int64 {
	p, err := state.Path(root, writesName)
	if err != nil {
		return 0
	}
	n := Writes(root) + 1
	if err := state.Write(p, []byte(fmt.Sprintf("%d\n", n)), 0o600); err != nil {
		return 0
	}
	return n
}
