package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
)

// splitSpecPieces is how many arguments after a spec that did not parse are
// tried as the rest of it: a pattern of more words than this is not looked for.
const splitSpecPieces = 8

// splitSpec returns err with a sentence added when args[i], which did not
// parse, parses as an address once the arguments after it are joined to it by
// single spaces: an unquoted `f.go:/func Foo/` reaches mrw as `f.go:/func` and
// `Foo/`, and the refusal of the first names a pattern nobody typed. It tries
// the spec and never matches on the error's words; a spec that stays broken
// within splitSpecPieces arguments keeps the refusal it had. The spec is
// printed in single quotes, which is quoting in a POSIX shell and in
// PowerShell; a Windows build may be answering cmd.exe, where it is not, and a
// spec that holds a single quote needs its shell's own escape, so neither is
// printed.
func splitSpec(args []string, i int, err error) error {
	joined := args[i]
	for j := i + 1; j < len(args) && j <= i+splitSpecPieces; j++ {
		joined += " " + args[j]
		if _, perr := read.ParseSpec(joined); perr != nil {
			continue
		}
		if runtime.GOOS == "windows" || strings.Contains(joined, "'") {
			return fmt.Errorf("%w; the shell split this spec at a space: quote the whole spec", err)
		}
		return fmt.Errorf("%w; the shell split this spec at a space: quote it, '%s'", err, joined)
	}
	return err
}
