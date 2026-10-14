package read

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// window is text cut to width characters (ADR-137): centred on the first match
// of any of res, or from the start when none matches or none is given, clamped
// to the line. "…" marks each side where text was cut, and a marker names the
// columns shown, 1-based and inclusive, and the length of the line. It is cut
// on runes, never inside a character.
func window(text string, width int, res []*regexp.Regexp) string {
	total := utf8.RuneCountInString(text)
	first := -1
	for _, re := range res {
		if re == nil {
			continue
		}
		if loc := re.FindStringIndex(text); loc != nil {
			if c := utf8.RuneCountInString(text[:loc[0]]); first < 0 || c < first {
				first = c
			}
		}
	}
	start := 0
	if first >= 0 {
		start = first - width/2
	}
	start = max(start, 0)
	end := start + width
	if end > total {
		end = total
		start = max(end-width, 0)
	}
	var b strings.Builder
	if start > 0 {
		b.WriteString("…")
	}
	// Slice the original string at the window's byte offsets: converting the
	// whole line to runes would allocate four times its length to show a few
	// dozen characters (the Codex review of #367).
	from := runeOffset(text, start)
	b.WriteString(text[from : from+runeOffset(text[from:], end-start)])
	if end < total {
		b.WriteString("…")
	}
	fmt.Fprintf(&b, "  [cols %d-%d of %d]", start+1, end, total)
	return b.String()
}

// runeOffset is the byte offset in s of its n-th character (0-based), or len(s)
// when s has fewer; it allocates nothing.
func runeOffset(s string, n int) int {
	for i := range s {
		if n == 0 {
			return i
		}
		n--
	}
	return len(s)
}
