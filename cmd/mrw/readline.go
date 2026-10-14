package main

import (
	"fmt"
	"strings"
)

// noMatchLine is the last line of a read whose search found no file (ADR-146).
// It says `no match:` when nothing the caller asked for was refused, and
// `incomplete:` when part of the search was never made because a named path
// was. The pattern is shown on one line, and an ADR-140 hint, when there is
// one, ends the line in both kinds.
func noMatchLine(pattern string, refused int, hint string) string {
	found := fmt.Sprintf("no file matched /%s/", oneLine(pattern))
	if refused > 0 {
		return fmt.Sprintf("incomplete: %s, and %d path(s) were refused (see the REFUSED lines)%s", found, refused, hint)
	}
	return "no match: " + found + hint
}

// oneLine spells a pattern's line breaks, so the kind that begins the last line
// is on the last line a caller reads.
func oneLine(s string) string {
	return strings.NewReplacer("\r", `\r`, "\n", `\n`).Replace(s)
}
