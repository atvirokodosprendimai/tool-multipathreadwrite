package apply

import (
	"fmt"
	"regexp"
	"strings"
)

// closerToken matches a line that closes a structure in the stacks the field
// reports measured (ADR-119): a markdown fence, a run of closing brackets with
// their trailing punctuation, an HTML or XML end tag, a Blade @end directive,
// and the shell and Ruby/Lua block ends. Only such a line leaves a duplicate
// closer behind; a repeated ordinary line is usually a repeated statement.
var closerToken = regexp.MustCompile("^(`{3,}|~{3,}|[}\\])]+[;,)]*|</[\\w.-]+>|@end\\w*|end|fi|done|esac)$")

// closerWindow is how many non-blank lines after a body the closer hint looks
// at. The markdown field report's surviving fence was the fourth; the window
// was registered in BACKLOG before the measurement that judged it (ADR-119).
const closerWindow = 4

// closerHint reports the line after a replace's body that repeats the body's
// last non-blank line, when that line is a closer: the closer the file already
// had, left below a body that brought its own. after is the written file from
// the line after the body, and first is that line's number. Empty when the body
// does not end in a closer, or none of the next closerWindow non-blank lines
// repeats it. Advice only: the hunk stays ok (ADR-119).
func closerHint(body, after []string, first int) string {
	last := ""
	for i := len(body) - 1; i >= 0 && last == ""; i-- {
		last = strings.TrimSpace(body[i])
	}
	if last == "" || !closerToken.MatchString(last) {
		return ""
	}
	looked := 0
	for i, line := range after {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if t == last {
			return fmt.Sprintf("line %d repeats the body's last line: %s", first+i, t)
		}
		if looked++; looked == closerWindow {
			break
		}
	}
	return ""
}

// clearWriteRows drops what a hunk says about a write that did not happen:
// the pad (ADR-052), the balance row (ADR-054) and the closer hint (ADR-119).
// Every site that turns an ok hunk into a skipped or failed one calls it, so a
// row added later is cleared at all of them at once.
func (h *HunkResult) clearWriteRows() {
	h.Echo = nil
	h.Balance = ""
	h.Closer = ""
}
