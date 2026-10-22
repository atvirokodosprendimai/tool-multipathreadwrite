package read

import "testing"

// Issue #496, review of 645c6774. The command read prints for a file outside the
// root is pasted into a shell, so a name the shell would take for syntax is
// quoted: a backslash (Git Bash strips it from an unquoted path) and a space
// alike, and a single quote the way the shell it runs under spells it.
func TestAWordInTheSuggestedCommandIsQuotedForTheShellItRunsUnder(t *testing.T) {
	for _, c := range []struct {
		in      string
		windows bool
		want    string
	}{
		{"hosts:1", false, "hosts:1"},
		{"/tmp/other", false, "/tmp/other"},
		{"--help:1", false, "--help:1"},
		{"my file.txt:1", false, "'my file.txt:1'"},
		{" lead:1", false, "' lead:1'"},
		{"it's", false, `'it'\''s'`},
		{"it's", true, "'it''s'"},
		{`C:\Users\zy\data`, true, `'C:\Users\zy\data'`},
		{`C:\Users\zy\data`, false, `'C:\Users\zy\data'`},
		{"", false, "''"},
	} {
		if got := quoteWord(c.in, c.windows); got != c.want {
			t.Errorf("quoteWord(%q, windows=%v) = %s; want %s", c.in, c.windows, got, c.want)
		}
	}
}
