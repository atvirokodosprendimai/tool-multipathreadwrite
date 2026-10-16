package seen

import (
	"path/filepath"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// Dir is the state directory that holds root's ledger and write lock: the
// checkout's, which is root's own only when root is the checkout's top.
func Dir(root string) (string, error) {
	top, _ := state.Checkout(root)
	return state.Dir(top)
}

// storedKey is a ledger key as the checkout's ledger holds it: root's path
// below the top joined to it. Every key is root-relative — the boundary refuses
// a path outside the root, and a read of an absolute path inside it is keyed
// relative — so there is no absolute key to pass through.
func storedKey(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return filepath.Join(prefix, key)
}

// within is the checkout's ledger as root sees it: the relative keys under
// prefix, without it. The rest of the checkout stays in the file; Record and
// Drop work on the whole ledger, never on this view.
func within(l Ledger, prefix string) Ledger {
	if prefix == "" {
		return l
	}
	out := Ledger{}
	below := prefix + string(filepath.Separator)
	for k, o := range l {
		if rest, ok := strings.CutPrefix(k, below); ok {
			out[rest] = o
		}
	}
	return out
}
