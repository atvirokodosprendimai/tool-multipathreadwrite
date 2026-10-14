package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// migrateLegacyState copies any pre-ADR-004 in-tree state in ./.mrw/ into the
// state directory, once, and announces it on stderr: a tool that quietly moves
// your files is the sibling of the tool that quietly created them. It runs from
// the root command's Before (ADR-145), so the parser has already answered the
// version flag in every spelling it takes, --help and a flag-parse error: none
// of those reaches here, and none touches state.
func migrateLegacyState() {
	moved, err := state.Migrate(".")
	if err != nil || len(moved) == 0 {
		return
	}
	if dir, err := state.Dir("."); err == nil {
		fmt.Fprintf(os.Stderr, "mrw: moved %s from ./%s/ to %s — the copy in your working tree is "+
			"untouched and can now be deleted\n", strings.Join(moved, " and "), state.LegacyDir, dir)
	}
}
