// Package addr holds the LEXICAL half of mrw's address grammar: the part a
// read spec and a plan hunk must agree on to the byte.
//
// Resolution stays split, and deliberately so — internal/read serves lines,
// internal/plan addresses a hunk, and only internal/apply knows how long a file
// is. What lives here is what a caller TYPES: recognising a relative end,
// validating its digits, and the exact wording of every refusal.
//
// WHY IT IS SHARED RATHER THAN DUPLICATED. ADR-026 first shipped this logic
// twice, once per parser, with a contract row as the anti-drift gate. A review
// of PR #125 found the copies had already drifted before the branch merged:
// `f.txt:,+3` served lines 1-4 at exit 0 on the read path while the plan path
// refused the identical string as an empty address. docs/adr/BACKLOG.md's "One
// address parser, not two" pre-registered the trigger — worth a record if the
// divergence happens twice — and this was the second time. A contract row can
// only notice a drift after someone writes the case; one function cannot drift.
package addr

import (
	"fmt"
	"strconv"
	"strings"
)

// CutRelative splits a relative end — `A,+N`, the N lines after A (ADR-026) —
// off an address, returning the base for the caller's own parser and N. n is 0
// when the address carries no relative end, and base is then s unchanged.
//
// Every base that is not a SINGLE start is refused here rather than in either
// parser, because a relative end REPLACES the end: `5-7,+3` would otherwise
// discard the 7 the caller wrote and report success for a span they did not
// ask for.
func CutRelative(s string) (base string, n int, err error) {
	if strings.HasPrefix(s, "+") {
		return "", 0, fmt.Errorf("%q has no start to be relative to: write %s for that line, "+
			"or A,%s for the %s lines after A", s, s[1:], s, s[1:])
	}
	k := strings.LastIndex(s, ",+")
	if k < 0 || !isDigits(s[k+2:]) {
		return s, 0, nil
	}
	n, convErr := strconv.Atoi(s[k+2:])
	if convErr != nil || n < 1 {
		return "", 0, fmt.Errorf("bad relative end %q in %q: write ,+N with N at least 1 for the "+
			"N lines after the start, or drop it to address the start alone", s[k+1:], s)
	}
	base = s[:k]
	switch {
	case base == "":
		return "", 0, fmt.Errorf("%q has no start to be relative to: write the line or pattern "+
			"the count runs from, as in 12%s", s, s[k:])
	case base == "0" || base == "-":
		return "", 0, fmt.Errorf("%q cannot carry a relative end: %s names no line to count from", s, base)
	case strings.HasPrefix(base, "/"):
		// A pattern may contain "-" and ",", so only the two-pattern form is
		// checked here; everything else about the pattern is the caller's to
		// parse.
		if strings.Contains(base, "/,/") {
			return "", 0, fmt.Errorf("%q has both an end pattern and a relative end: "+
				"write /from/,/to/ or A,+N, not both", s)
		}
	case strings.Contains(base, "-"):
		return "", 0, fmt.Errorf("%q already has an end (%s): a relative end replaces it, "+
			"so write one or the other", s, base)
	}
	return base, n, nil
}

// isDigits reports whether t is one or more ASCII digits. It is what keeps a
// `,+` INSIDE a pattern from being read as a relative end: `/a,+3/` ends in
// "3/", which is not a number, so the suffix is left alone.
func isDigits(t string) bool {
	if t == "" {
		return false
	}
	for i := 0; i < len(t); i++ {
		if t[i] < '0' || t[i] > '9' {
			return false
		}
	}
	return true
}
