package main

import "fmt"

// noMatchLine is the last line of a read whose search found no file (ADR-146).
// It says `no match:` when nothing the caller asked for was refused, and
// `incomplete:` when part of the search was never made because a named path
// was.
func noMatchLine(found string, refused int) string {
	if refused > 0 {
		return fmt.Sprintf("incomplete: %s, and %d path(s) were refused (see the REFUSED lines)", found, refused)
	}
	return "no match: " + found
}
