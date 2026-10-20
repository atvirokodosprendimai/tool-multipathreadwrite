// Package preview shows what a plan maker's dry run will change (ADR-164
// Decision 2). It is a leaf: it takes the lines a hunk replaces and the lines
// that replace them, and returns the rows a receipt prints under the hunk's
// ok row. Line endings are not shown; the lines are the split internal/lines
// makes, terminators removed.
package preview

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	context = 40  // characters kept each side of a marked span
	spanMax = 160 // a span longer than this keeps only its two ends
	spanEnd = 70  // characters kept at each end of a long span
	lineMax = 200 // a line with nothing to mark is cut to this many characters
)

// Lines renders one replace hunk. One line replaced by one line is a single
// `~` row with the change marked in place as [-old-]{+new+}. Anything else is
// a `-` row for each replaced line, then a `+` row for each body line, a line
// paired with the line at the same index on the other side marked the same
// way and a line with no counterpart cut to its first 200 characters. The
// marked span runs from the first to the last byte that differs, so every
// change on a line shows however far apart, and a span over 160 characters
// keeps its first and last 70 with the count between. firstLine numbers the
// first row of each side.
func Lines(before, after []string, firstLine int) []string {
	if len(before) == 1 && len(after) == 1 {
		return []string{fmt.Sprintf("   ~ %d| %s", firstLine, mark(before[0], after[0], true, true))}
	}
	var out []string
	for i, b := range before {
		text := head(b)
		if i < len(after) {
			text = mark(b, after[i], true, false)
		}
		out = append(out, fmt.Sprintf("   - %d| %s", firstLine+i, text))
	}
	for i, a := range after {
		text := head(a)
		if i < len(before) {
			text = mark(before[i], a, false, true)
		}
		out = append(out, fmt.Sprintf("   + %d| %s", firstLine+i, text))
	}
	return out
}

// mark returns old's line (showOld) and/or new's, the differing span between
// them in [-…-] and {+…+} and context round it. An empty side is not marked.
func mark(old, new string, showOld, showNew bool) string {
	p := 0
	for p < len(old) && p < len(new) && old[p] == new[p] {
		p++
	}
	for p > 0 && (p < len(old) && !utf8.RuneStart(old[p]) || p < len(new) && !utf8.RuneStart(new[p])) {
		p--
	}
	s := 0
	for s < len(old)-p && s < len(new)-p && old[len(old)-1-s] == new[len(new)-1-s] {
		s++
	}
	for s > 0 && (!utf8.RuneStart(old[len(old)-s]) || !utf8.RuneStart(new[len(new)-s])) {
		s--
	}
	if p == len(old) && p == len(new) {
		return head(old) // identical: nothing to mark
	}
	line := old
	if !showOld {
		line = new
	}
	var b strings.Builder
	b.WriteString(tail(line[:p], context))
	if showOld && p < len(old)-s {
		b.WriteString("[-" + span(old[p:len(old)-s]) + "-]")
	}
	if showNew && p < len(new)-s {
		b.WriteString("{+" + span(new[p:len(new)-s]) + "+}")
	}
	b.WriteString(front(line[len(line)-s:], context))
	return b.String()
}

// span is a marked span, its first and last 70 characters when over 160.
func span(s string) string {
	r := []rune(s)
	if len(r) <= spanMax {
		return s
	}
	return string(r[:spanEnd]) + fmt.Sprintf("…[%d chars]…", len(r)-2*spanEnd) + string(r[len(r)-spanEnd:])
}

// head is a line with nothing marked, cut to its first 200 characters.
func head(s string) string {
	if utf8.RuneCountInString(s) <= lineMax {
		return s
	}
	return string([]rune(s)[:lineMax]) + "…"
}

// tail is the last n characters of s, led by … when s was longer.
func tail(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return "…" + string(r[len(r)-n:])
}

// front is the first n characters of s, followed by … when s was longer.
func front(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
