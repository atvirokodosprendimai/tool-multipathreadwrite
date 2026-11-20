package read

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Sub turns a --grep walk into a plan instead of lines (ADR-161): each served
// line is rewritten with Re.ReplaceAllString, and the lines that change are the
// plan's hunks. Run fills Plan, Sites, Files and Refused; a caller reads them
// after the run. It is handed to Run by pointer and not copied.
type Sub struct {
	Re   *regexp.Regexp
	Repl string
	// Plan is the plan text: one `replace` hunk per changed line.
	Plan strings.Builder
	// Sites and Files count the hunks and the files holding them.
	Sites, Files int
	// Unanchored counts the hunks written with no anchor=, because their line has
	// no piece an anchor can name (ADR-203): empty, or nothing but CR and invalid
	// UTF-8. A plan written by a later call is refused on those until read.
	Unanchored int
	// Refused names each file whose path the plan grammar cannot carry. When
	// any is named, Run plans and licenses nothing at all.
	Refused []string
}

// file plans one file's served spans and returns the lines it planned, which
// are the only lines the read licenses. A line the rewrite leaves unchanged is
// neither planned nor licensed. The slice is never nil: a file with nothing
// planned is hashed, and none of it shown. Every hunk carries sha, the content
// the plan was computed from, so it applies to that version or to none.
func (s *Sub) file(path, sha string, lns []string, spans []span) [][2]int {
	planned := [][2]int{}
	name, ok := planPath(path)
	if !ok {
		s.Refused = append(s.Refused, fmt.Sprintf("%q: a line break in a path cannot be written in a plan header", path))
		return planned
	}
	for _, sn := range spans {
		for ln := sn.start; ln <= sn.end; ln++ {
			text := lns[ln-1]
			// Split removed the file's terminator, so a CRLF file's text has no CR
			// and the plan body carries none.
			next := s.Re.ReplaceAllString(text, s.Repl)
			if next == text {
				continue
			}
			// A plan line's trailing CR is dropped by the parser, so a new text
			// ending in one (a line of a file whose endings are mixed) would be
			// written a byte short. Refused, and the whole plan with it.
			if strings.HasSuffix(next, "\r") {
				s.Refused = append(s.Refused, fmt.Sprintf("%s:%d: the new text ends in a CR byte, which a plan line cannot carry (a file whose line endings are mixed)", ShownPath(path), ln))
				return [][2]int{}
			}
			// A counted raw body whatever the text: one that starts with @@, body=
			// or a byte order mark must not be read as a header or an option.
			hdr := fmt.Sprintf("@@ %s %d replace sha=%s body=1 raw=true", name, ln, sha)
			// anchor= is a piece of the line as it was, so a plan written by a later
			// call is not refused as a number alone (ADR-203).
			if a, ok := anchorOf(text); ok {
				hdr += ` anchor="` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(a) + `"`
			} else {
				s.Unanchored++
			}
			s.Plan.WriteString(hdr + "\n" + next + "\n")
			s.Sites++
			planned = append(planned, [2]int{ln, ln})
		}
	}
	if len(planned) > 0 {
		s.Files++
	}
	return planned
}

// anchorMax bounds an anchor: the check is a substring match, so a piece of a
// line says as much as the line, and a 17 MiB line must not push a header past
// the plan parser's limit.
const anchorMax = 200

// anchorOf is the piece of text a hunk's anchor= names: the first non-empty
// stretch of valid UTF-8 between CR bytes and invalid bytes, cut to anchorMax
// bytes at a rune boundary. It reports false when no stretch qualifies (an
// empty line, or nothing but CR and invalid bytes).
func anchorOf(text string) (string, bool) {
	from := 0
	for i := 0; i <= len(text); {
		end, size := i == len(text), 0
		if !end {
			r, n := utf8.DecodeRuneInString(text[i:])
			size = n
			end = r == '\r' || (r == utf8.RuneError && n == 1)
		}
		if !end {
			i += size
			continue
		}
		if seg := text[from:i]; seg != "" {
			if len(seg) > anchorMax {
				n := anchorMax
				for n > 0 && !utf8.RuneStart(seg[n]) {
					n--
				}
				seg = seg[:n]
			}
			return seg, true
		}
		i += max(size, 1)
		from = i
	}
	return "", false
}

// planPath spells path as the first field of a plan header. The header splitter
// ends a field at a space or tab and reads a double quote and a backslash as
// escapes, so a path holding any of them is double-quoted with those two
// escaped; a single quote is literal there. A path starting with @ would be read
// as a pointer into the working set (iter.IsPointer), so it is spelled ./@…,
// which names the same file. Only a line break cannot be carried, because a
// header is one line.
func planPath(path string) (string, bool) {
	p := filepath.ToSlash(path)
	if strings.ContainsAny(p, "\r\n") {
		return "", false
	}
	if strings.HasPrefix(p, "@") {
		p = "./" + p
	}
	if !strings.ContainsAny(p, " \t\"'\\") {
		return p, true
	}
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(p) + `"`, true
}
