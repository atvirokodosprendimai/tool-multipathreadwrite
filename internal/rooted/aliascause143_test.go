package rooted

import (
	"strings"
	"testing"
)

// The refusal of a name Windows does not keep names the cause that applies, not
// all three, and no longer tells the caller to "name the file as it is on disk"
// when the file on disk really has that name (the Windows retest of v1.60.0).
func TestAnAliasRefusalNamesTheCauseThatApplies(t *testing.T) {
	for _, c := range []struct{ comp, want string }{
		{"trail.", "trailing dot or space"},
		{"trail ", "trailing dot or space"},
		{"a:b", "NTFS stream"},
		{"\xff.txt", "not valid UTF-8"},
	} {
		got := aliasCause(c.comp)
		if !strings.Contains(got, c.want) {
			t.Errorf("aliasCause(%q) = %q, want it to name %q", c.comp, got, c.want)
		}
		for _, other := range []string{"trailing dot or space", "NTFS stream", "not valid UTF-8"} {
			if other != c.want && strings.Contains(got, other) {
				t.Errorf("aliasCause(%q) = %q names another cause (%q) too", c.comp, got, other)
			}
		}
		if strings.Contains(got, "as it is on disk") {
			t.Errorf("aliasCause(%q) = %q still sends the caller to the name on disk", c.comp, got)
		}
	}
}
