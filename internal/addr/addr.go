// Package addr holds the LEXICAL half of mrw's address grammar: the part a
// read spec and a plan hunk must agree on to the byte.
//
// Resolution stays split, and deliberately so — internal/read serves lines,
// internal/plan addresses a hunk, and only internal/apply knows how long a file
// is. What lives here is what a caller TYPES: recognising a relative end,
// validating its digits, proving the base is ONE start, and the exact wording of
// every refusal.
//
// WHY IT IS SHARED RATHER THAN DUPLICATED. ADR-026 first shipped this logic
// twice, once per parser, with a contract row as the anti-drift gate. A review
// of PR #125 found the copies had already drifted before the branch merged:
// `f.txt:,+3` served lines 1-4 at exit 0 on the read path while the plan path
// refused the identical string. docs/adr/BACKLOG.md's "One address parser, not
// two" pre-registered the trigger — worth a record if the divergence happens
// twice — and this was the second time. A contract row can only notice a drift
// after someone writes the case; one function cannot drift.
//
// ⚠ IT SCANS RATHER THAN SEARCHES. The first cut used strings.Contains to spot
// the two-pattern form, which refused the legal base `/a\/,/` — an escaped
// slash followed by a comma reads as the `/,/` delimiter to a substring search.
// A second review of the same PR found it, along with `/a/,+2,+3`, where cutting
// the last suffix leaves a base each parser then read differently. Delimiters
// are scanned with escapes honoured, and the base must be exactly one start.
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
// discard the 7 the caller wrote and report success for a span they did not ask
// for.
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
		// A count too large for an int lands here too, and "write ,+N with N at
		// least 1" is the right thing to say about it: the file that could
		// satisfy it does not exist.
		return "", 0, fmt.Errorf("bad relative end %q in %q: write ,+N with N at least 1 for the "+
			"N lines after the start, or drop it to address the start alone", s[k+1:], s)
	}
	if err := singleStart(s[:k], s); err != nil {
		return "", 0, err
	}
	return s[:k], n, nil
}

// singleStart reports whether base names exactly one starting line. whole is the
// address as the caller wrote it, so every refusal can quote that rather than
// the fragment left after the cut.
func singleStart(base, whole string) error {
	switch {
	case base == "":
		return fmt.Errorf("%q has no start to be relative to: write the line or pattern "+
			"the count runs from, as in 12%s", whole, whole[strings.LastIndex(whole, ",+"):])
	case base == "0" || base == "-":
		return fmt.Errorf("%q cannot carry a relative end: %s names no line to count from", whole, base)
	case strings.HasPrefix(base, "/"):
		end := ClosingDelim(base, 0)
		if end < 0 {
			return fmt.Errorf("%q has an unclosed pattern: expected a second /", whole)
		}
		switch rest := base[end+1:]; {
		case rest == "":
			return nil
		case strings.HasPrefix(rest, ",/"):
			return fmt.Errorf("%q has both an end pattern and a relative end: "+
				"write /from/,/to/ or A,+N, not both", whole)
		default:
			return fmt.Errorf("%q has %q after the pattern: a relative end takes one start, "+
				"so write the pattern or the extra part, not both", whole, rest)
		}
	case base == "$":
		return nil
	case !isDigits(base):
		// `5-7`, `5-`, `$-5` and `5,+1` all land here: each already names an
		// end, and a relative end replaces an end rather than joining one.
		return fmt.Errorf("%q already has an end (%s): a relative end replaces it, "+
			"so write one or the other", whole, base)
	}
	return nil
}

// ClosingDelim returns the index of the slash that closes a pattern opened at
// index `open`, or -1 when it is never closed. It is exported because all three
// scanners in this repository — this package, internal/read's splitRanges and
// internal/plan's parsePattern — must agree byte for byte about where a pattern
// ends; two of them disagreeing is how `/a\/,/` became "both an end pattern and
// a relative end".
//
// ⚠ IT COUNTS BACKSLASH PARITY, and `s[i-1] != '\\'` does not. In `/\\/` the
// pattern is a single literal backslash and the final slash CLOSES it, because
// the backslash before it is itself escaped. All three scanners tested only the
// preceding byte, so that address was read as unclosed and a legal regexp was
// refused. Found by the third Codex review of PR #125.
func ClosingDelim(s string, open int) int {
	for i := open + 1; i < len(s); i++ {
		if s[i] != '/' {
			continue
		}
		back := 0
		for j := i - 1; j > open && s[j] == '\\'; j-- {
			back++
		}
		if back%2 == 0 {
			return i
		}
	}
	return -1
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
