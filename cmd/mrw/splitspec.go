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

// splitSpec returns err with a sentence added when spec, which did not parse,
// is an argument of the command line whose pattern address the arguments after
// it complete: an unquoted `f.go:/func Foo/` reaches mrw as `f.go:/func` and
// `Foo/`, and the refusal of the first names a pattern nobody typed.
//
// It is narrow on purpose. Every argument must be a spec as typed: with a
// working-set number among them (`@1`), which argument a failing spec came
// from is not known here, so nothing is said, and a --files-from line was not
// split by a shell (no argument is typed beside a list). Joined to the
// arguments after it by single spaces the spec must parse to a path it
// already began with, PATH:/…, with pattern ranges only, so a join that
// merely parses as some other file or address says nothing. It tries the spec
// and never matches on the error's words.
//
// The spec is printed, in single quotes, only when every byte of it is
// printable ASCII other than a single quote and it does not start with a
// dash: that is one word in a POSIX shell and in PowerShell, and nothing a
// terminal acts on. A Windows build may be answering cmd.exe, where single
// quotes do not quote, so it prints no spelling either.
func splitSpec(typed []string, spec string, err error) error {
	at := -1
	for i, a := range typed {
		if strings.HasPrefix(a, "@") {
			return err
		}
		if a == spec && at < 0 { // the first: an earlier one like it would have stopped the read
			at = i
		}
	}
	if at < 0 {
		return err
	}
	joined := spec
	for j := at + 1; j < len(typed) && j <= at+splitSpecPieces; j++ {
		joined += " " + typed[j]
		sp, perr := read.ParseSpec(joined)
		if perr != nil || !strings.HasPrefix(spec, sp.Path+":/") || !patternsOnly(sp.Ranges) {
			continue
		}
		if runtime.GOOS == "windows" || !plainWord(joined) {
			return fmt.Errorf("%w; the shell split this spec at a space: quote the whole spec", err)
		}
		return fmt.Errorf("%w; the shell split this spec at a space: quote it, '%s'", err, joined)
	}
	return err
}

// patternsOnly reports whether there is a range and every range starts at a pattern.
func patternsOnly(rs []read.Range) bool {
	for _, r := range rs {
		if r.Re == nil {
			return false
		}
	}
	return len(rs) > 0
}

// plainWord reports whether s can be printed between single quotes as it is.
func plainWord(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e || s[i] == '\'' {
			return false
		}
	}
	return !strings.HasPrefix(s, "-")
}
