package read

import (
	"strconv"
	"testing"
)

// ShownPath is injective: no two paths print alike, so a header names one file.
// A path that is printed quoted is the only kind that begins with a quote, so an
// escaped name cannot equal a file literally named with the quote characters.
func TestShownPathIsInjective(t *testing.T) {
	paths := []string{"", "a", "a b", `"a"`, `"a\nb"`, "a\nb", `a\nb`, "\"", `\`, "a\r", "a\x00", "\x7f", `"draft\nnotes.txt"`, "draft\nnotes.txt", `"\"a\""`}
	seen := map[string]string{}
	for _, p := range paths {
		s := ShownPath(p)
		if prev, dup := seen[s]; dup {
			t.Errorf("%q and %q are both shown as %s", prev, p, s)
		}
		seen[s] = p
		if s != p {
			if u, err := strconv.Unquote(s); err != nil || u != p {
				t.Errorf("%q is shown as %s, which does not read back", p, s)
			}
		} else if len(s) > 0 && s[0] == '"' {
			t.Errorf("%q is shown unquoted though it begins with a quote", p)
		}
	}
}
